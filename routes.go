package main

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Index,
	},
	Route{
		"VideoToImage",
		"POST",
		"/image_sets",
		ConvertVideoToImage,
	},
	Route{
		"ImageSet",
		"GET",
		"/image_sets/{setId}",
		GetImageSet,
	},
	Route{
		"CheckIfDone",
		"GET",
		"/image_sets/{setId}/IsDone",
		CheckIfDone,
	},
}
