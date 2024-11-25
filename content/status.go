package content

// Status status type SiteContent respnses
type Status int

const (
	// StatusOk we found content
	StatusOk Status = 200
	// StatusForbidden we found content but you mst not access it
	StatusForbidden = 403
	// StatusNotFound we did not find content
	StatusNotFound = 404
)
