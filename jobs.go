package main

type recordJob struct {
	record Record
}

type closeWriterJob struct {
	path string
}
