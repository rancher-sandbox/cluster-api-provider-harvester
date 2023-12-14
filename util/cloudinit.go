package util

import (
	"fmt"

	"golang.org/x/exp/slices"

	"emperror.dev/errors"
	"gopkg.in/yaml.v2"
)

var cloudInitListSections = []string{"packages", "runcmd", "ssh_authorized_keys", "groups", "users", "write_files", "bootcmd"}

func MergeCloudInitData(cloudInits ...string) ([]byte, error) {
	var resultCloudInit []byte

	// resCloudInitObj will be an object that stores the result of merging all the cloud-init objects
	resCloudInitObj := make(map[string]interface{})

	// First of all we iterate over all the cloud-init objects
	for _, cloudInit := range cloudInits {
		cloudInitObj := make(map[string]interface{})
		err := yaml.Unmarshal([]byte(cloudInit), &cloudInitObj)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshall cloud-init, input cloud-init is malformed: %v", err)
		}

		// For each cloud-init object we iterate over all the keys and values
		for k, v := range cloudInitObj {

			// If the key is a list section, we append the values to the resulting list key
			if slices.Contains(cloudInitListSections, k) {
				listSection := []interface{}{}

				// Get the current list section from the resulting cloud-init object if it exists
				if resCloudInitObj[k] != nil {
					listSection = resCloudInitObj[k].([]interface{})
				}

				// Append the values to the resulting list section
				for _, ind_value := range v.([]interface{}) {
					listSection = append(listSection, ind_value)
				}

				// Add the resulting list section to the resulting cloud-init object in current key
				resCloudInitObj[k] = listSection
			} else {
				// If the key is not a list section, we just add the key and value to the resulting cloud-init object
				resCloudInitObj[k] = v
			}
		}
	}

	resultCloudInit, err := yaml.Marshal(resCloudInitObj)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshall cloud-init, input cloud-init is malformed")
	}

	resultCloudInit = []byte("#cloud-config\n" + string(resultCloudInit))

	return resultCloudInit, nil
}
