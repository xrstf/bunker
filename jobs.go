package main

type recordJob struct {
	tag    string
	record *Record
}

type closeWriterJob struct {
	path string
}
