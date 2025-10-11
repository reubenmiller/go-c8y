package c8y_api

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// DecodeJSONBytes decodes json preserving number formatting (especially large integers and scientific notation floats)
func DecodeJSONBytes(v []byte, dst interface{}) error {
	return DecodeJSONReader(bytes.NewReader(v), dst)
}

// DecodeJSONFile decodes a json file into dst interface
func DecodeJSONFile(filepath string, dst interface{}) error {
	fp, err := os.Open(filepath)
	if err != nil {
		return err
	}

	defer fp.Close()
	buf, err := io.ReadAll(fp)
	if err != nil {
		return err
	}
	return DecodeJSONReader(bytes.NewReader(buf), dst)
}

// DecodeJSONReader decodes bytes using a reader interface
//
// Note: Decode with the UseNumber() set so large or
// scientific notation numbers are not wrongly converted to integers!
// i.e. otherwise this conversion will happen (which causes a problem with mongodb!)
//
//	9.2233720368547758E+18 --> 9223372036854776000
func DecodeJSONReader(r io.Reader, dst interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(&dst)
}
