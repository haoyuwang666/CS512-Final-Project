// TODO(Haoyu): Freeze these DTOs with both consumers before adding fields or endpoints.
package protocol

import "cooperative.video/internal/domain"

type CreateVideo struct {
	OperationID string          `json:"operation_id"`
	Manifest    domain.Manifest `json:"manifest"`
}
type ConfirmReplica struct {
	OperationID string         `json:"operation_id"`
	Receipt     domain.Receipt `json:"receipt"`
}
type PublishVideo struct {
	OperationID     string `json:"operation_id"`
	ExpectedVersion int    `json:"expected_version"`
}
type Locations struct {
	SegmentID string            `json:"segment_id"`
	Locations []domain.Location `json:"locations"`
}
type SessionRequest struct {
	VideoID  string `json:"video_id"`
	Strategy string `json:"strategy"`
}
