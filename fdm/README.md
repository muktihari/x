# File Download Manager

[![expr](https://img.shields.io/badge/go-reference-blue.svg?style=flat)](https://pkg.go.dev/github.com/muktihari/x/fdm)

IDM-like, just in terminal and without pause/resume capability :)

![Alt Text](./preview.gif)

It's all about working with bytes, splitting request using "range" header and writing the bytes of response to at certain point of a file's bytes (as it satisfies io.Seeker). Exploring io.Writer and io.Reader and how we can intercept the interface to print the progress. That's it!