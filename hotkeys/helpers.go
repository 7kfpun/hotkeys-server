package main

import (
	"encoding/csv"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)


func Unmarshal(reader *csv.Reader, v interface{}) error {
	record, err := reader.Read()
	if err != nil {
		return err
	}
	s := reflect.ValueOf(v).Elem()
	if s.NumField() != len(record) {
		return &FieldMismatch{s.NumField(), len(record)}
	}
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		switch f.Type().String() {
		case "string":
			f.SetString(record[i])
		case "int":
			ival, err := strconv.ParseInt(record[i], 10, 0)
			if err != nil {
				return err
			}
			f.SetInt(ival)
		default:
			return &UnsupportedType{f.Type().String()}
		}
	}
	return nil
}


type FieldMismatch struct {
	expected, found int
}


func (e *FieldMismatch) Error() string {
	return "CSV line fields mismatch. Expected " + strconv.Itoa(e.expected) + " found " + strconv.Itoa(e.found)
}


type UnsupportedType struct {
	Type string
}


func (e *UnsupportedType) Error() string {
	return "Unsupported type: " + e.Type
}


func downloadFromUrl(ctx context.Context, url string) string {
	client := urlfetch.Client(ctx)
	response, err := client.Get(url)
	if err != nil {
		log.Errorf(ctx, "Error while downloading " + url, err)
		return ""
	}
	defer response.Body.Close()

	log.Infof(ctx, "Saw record %s", response.Body)
	log.Infof(ctx, "Saw record %v", response.Status)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf(ctx, "Error while reading" + url, err)
		return ""
	}
	log.Infof(ctx, "Saw record %s", body)

	return string(body)
}


func cleanUris(url string) []string {
	url = strings.Replace(url, "https://", "", -1)
	url = strings.Replace(url, "http://", "", -1)
	uris := strings.Split(url, "/")

	var cleaned_uris []string

	cleaned_uris = append(cleaned_uris, uris[0])

	if len(uris) > 1 {
		cleaned_uris = append(cleaned_uris, strings.Join(uris[:2], "/"))
	}

	return cleaned_uris
}
