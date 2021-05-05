package controllers

import (
	"bytes"
	"io/ioutil"
	"log"

	orkestrav1alpha1 "github.com/Azure/Orkestra/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

var bookinfoExampleFilePath string = "../examples/simple/bookinfo.yaml"

func bookinfo() *orkestrav1alpha1.ApplicationGroup {
	g := &orkestrav1alpha1.ApplicationGroup{
		ObjectMeta: v1.ObjectMeta{
			Name: "bookinfo",
		},
	}

	yamlFile, err := ioutil.ReadFile(bookinfoExampleFilePath)
	if err != nil {
		log.Fatalf("yamlFile.Get err #%v", err)
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(yamlFile), 100)
	err = decoder.Decode(g)
	if err != nil {
		log.Fatalf("Decode err #%v", err)
	}

	return g
}
