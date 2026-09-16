package store

type Record struct {
	Path string
}

func Load(name string) (Record, error) {
	return Record{Path: name}, nil
}
