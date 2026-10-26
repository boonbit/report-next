package report

import "encoding/json"

// MarshalJSON serializes the IR deterministically: struct field order is
// fixed and encoding/json sorts map keys, so the same report always
// produces the same bytes.
func MarshalJSON(rep Report) ([]byte, error) {
	return json.MarshalIndent(rep, "", "  ")
}

// UnmarshalJSON parses IR JSON back into a Report.
func UnmarshalJSON(data []byte) (Report, error) {
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return Report{}, err
	}
	return rep, nil
}
