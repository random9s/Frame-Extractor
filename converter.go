package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	. "github.com/3d0c/gmf"
)

func fatal(err error) {
	debug.PrintStack()
	log.Fatal(err)
}

func assert(i interface{}, err error) interface{} {
	if err != nil {
		fatal(err)
	}

	return i
}

var i int32 = 0

func writeFile(b []byte, dir string) {
	name := "./temps/" + dir + "/" + strconv.Itoa(int(atomic.AddInt32(&i, 1))) + ".jpg"

	fp, err := os.Create(name)
	if err != nil {
		fatal(err)
	}

	defer func() {
		if err := fp.Close(); err != nil {
			fatal(err)
		}
	}()

	if n, err := fp.Write(b); err != nil {
		fatal(err)
	} else {
		log.Println(n, "bytes written to", name)
	}
}

func encodeWorker(data chan *Frame, wg *sync.WaitGroup, srcCtx *CodecCtx, dir string) {
	defer wg.Done()
	log.Println("worker started")
	codec, err := FindEncoder(AV_CODEC_ID_JPEG2000)
	if err != nil {
		fatal(err)
	}

	cc := NewCodecCtx(codec)
	defer Release(cc)

	w, h := srcCtx.Width(), srcCtx.Height()

	cc.SetPixFmt(AV_PIX_FMT_RGB24).SetWidth(w).SetHeight(h)

	if codec.IsExperimental() {
		cc.SetStrictCompliance(FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err := cc.Open(nil); err != nil {
		fatal(err)
	}

	swsCtx := NewSwsCtx(srcCtx, cc, SWS_BICUBIC)
	defer Release(swsCtx)

	// convert to RGB, optionally resize could be here
	dstFrame := NewFrame().
		SetWidth(w).
		SetHeight(h).
		SetFormat(AV_PIX_FMT_RGB24)
	defer Release(dstFrame)

	if err := dstFrame.ImgAlloc(); err != nil {
		fatal(err)
	}

	for {
		srcFrame, ok := <-data
		if !ok {
			break
		}
		//		log.Printf("srcFrome = %p",srcFrame)
		swsCtx.Scale(srcFrame, dstFrame)

		if p, ready, _ := dstFrame.EncodeNewPacket(cc); ready {
			writeFile(p.Data(), dir)
		}
		Release(srcFrame)
	}

}

func Zip(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	info, err := os.Stat(source)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})

	doneZipping, err := os.Create(source + "/done.txt")
	if err != nil {
		return err
	}

	defer doneZipping.Close()

	return err
}

func VideoToImage(dir, filename string, c chan<- string) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	srcFileName := "./temps/" + dir + "/" + filename

	inputCtx := assert(NewInputCtx(srcFileName)).(*FmtCtx)
	defer inputCtx.CloseInputAndRelease()

	srcVideoStream, err := inputCtx.GetBestStream(AVMEDIA_TYPE_VIDEO)
	if err != nil {
		c <- fmt.Sprintf("%s: %s\n", "VideoToImage", err)
		log.Println("No video stream found in", srcFileName)
	}

	wg := new(sync.WaitGroup)

	dataChan := make(chan *Frame)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go encodeWorker(dataChan, wg, srcVideoStream.CodecCtx(), dir)
	}

	for packet := range inputCtx.GetNewPackets() {
		if packet.StreamIndex() != srcVideoStream.Index() {
			continue
		}

		ist := assert(inputCtx.GetStream(packet.StreamIndex())).(*Stream)

		for frame := range packet.Frames(ist.CodecCtx()) {
			dataChan <- frame.CloneNewFrame()
		}
		Release(packet)
	}
	fmt.Println("Compressing...")
	Zip("./temps/"+dir, "./temps/"+dir+".zip")
	fmt.Println("Done")

	close(dataChan)

	wg.Wait()

	c <- fmt.Sprintf("%s: %s\n", "VideoToImage", "Done")
}
