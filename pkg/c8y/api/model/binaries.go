package model

// Binary ManagedObject representation
type Binary struct {
	*ManagedObject

	ContentType string `json:"contentType,omitempty"`
	Length      int64  `json:"length,omitempty,omitzero"`
	IsBinary    any    `json:"c8y_IsBinary"`
}

type BinaryCollection struct {
	*BaseResponse

	ManagedObjects []Binary `json:"managedObjects"`
}
