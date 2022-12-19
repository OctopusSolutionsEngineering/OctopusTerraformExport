package model

type GeneralCollection[T any] struct {
	TotalResults   int
	ItemsPerPage   int
	NumberOfPages  int
	LastPageNumber int
	ItemType       string
	Items          []T
}
