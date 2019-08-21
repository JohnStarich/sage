package model

// Institution represents the connection and identification details for a financial institution
type Institution interface {
	Description() string
	FID() string
	Org() string
}

// BasicInstitution includes the bare minimum information to implement an institution
// Typically used for adding accounts via OFX imports
type BasicInstitution struct {
	InstDescription string
	InstFID         string
	InstOrg         string
}

// Org implements Institution
func (i BasicInstitution) Org() string {
	return i.InstOrg
}

// FID implements Institution
func (i BasicInstitution) FID() string {
	return i.InstFID
}

// Description implements Institution
func (i BasicInstitution) Description() string {
	return i.InstDescription
}
