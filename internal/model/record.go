package model

import (
	"encoding/json"
	"time"
)

// DataRecord is the normalized envelope for all ingested data.
type DataRecord struct {
	SourceID  string                 `json:"source_id"`
	Timestamp int64                  `json:"timestamp"`
	RecordID  string                 `json:"record_id"`
	Payload   map[string]interface{} `json:"payload"`
}

func NewDataRecord(sourceID, recordID string, payload map[string]interface{}) DataRecord {
	return DataRecord{
		SourceID:  sourceID,
		Timestamp: time.Now().UnixMilli(),
		RecordID:  recordID,
		Payload:   payload,
	}
}

func (r *DataRecord) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalDataRecord(data []byte) (*DataRecord, error) {
	var r DataRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
