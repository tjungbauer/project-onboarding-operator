package conversion

import "encoding/json"

// ViaJSON copies src into dst by JSON round-trip. apiVersion and kind are stripped so
// the destination type receives the correct GVK from the conversion webhook / scheme.
func ViaJSON(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	delete(body, "apiVersion")
	delete(body, "kind")
	data, err = json.Marshal(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
