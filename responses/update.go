package responses

// Update - information about an update
type Update struct {
	// did it work or not
	Success bool `json:"success"`
	// this is for humans
	ErrorMessage string `json:"errorMessage"`
	Stats        Stats  `json:"stats"`
}
