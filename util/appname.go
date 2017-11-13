package util

import (
	"encoding/json"
	"os"

	"github.com/solo-io/pkg/errors"
)

func GetAppName() (string, error) {
	var appInfo vcapInfo
	if err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &appInfo); err != nil {
		return "", errors.New("unmarshalling VCAP_APPLICATION info "+os.Getenv("VCAP_APPLICATION"), err)
	}
	return appInfo.ApplicationName + "-cf-app", nil
}

func GetAppMem() (int, error) {
	var appInfo vcapInfo
	if err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &appInfo); err != nil {
		return -1, errors.New("unmarshalling VCAP_APPLICATION info "+os.Getenv("VCAP_APPLICATION"), err)
	}
	return appInfo.Limits.Mem, nil
}

type vcapInfo struct {
	Limits struct {
		Fds  int `json:"fds"`
		Mem  int `json:"mem"`
		Disk int `json:"disk"`
	} `json:"limits"`
	ApplicationName    string      `json:"application_name"`
	ApplicationUris    []string    `json:"application_uris"`
	Name               string      `json:"name"`
	SpaceName          string      `json:"space_name"`
	SpaceID            string      `json:"space_id"`
	Uris               []string    `json:"uris"`
	Users              interface{} `json:"users"`
	ApplicationID      string      `json:"application_id"`
	Version            string      `json:"version"`
	ApplicationVersion string      `json:"application_version"`
}
