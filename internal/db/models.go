package db

type Card struct {
	ID   int
	Name string
	Set  string
	// ... other fields
}

type Deck struct {
	ID    int
	Name  string
	Owner string
	// ... other fields
}
