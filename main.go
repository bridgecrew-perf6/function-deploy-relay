package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func main() {
	faClient := web.NewAppsClient("cbce1d09-3ba7-4d48-b220-3e0e2059a9f8") // TODO: envs
	// faClient.RequestInspector = LogRequest()
	// faClient.ResponseInspector = LogResponse()

	// Auth.
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		faClient.Authorizer = authorizer
	}

	// List all Function Apps in resource group.
	faList, err := faClient.ListComplete(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	// Create empty slice for function names.
	functionapps := []string{}
	// Set the regex to search for resource group name.
	matched := regexp.MustCompile(`\"serverFarmId\":"\/subscriptions\/.*\/resourceGroups\/mxi-dev`)
	// Loop through all functions.
	for notDone := true; notDone; notDone = faList.NotDone() {
		data := faList.Value()
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Println("jsonData: ", string(jsonData))
		// If the resource group is found, add the function to the slice.
		if matched.MatchString(string(jsonData)) == true {
			functionapps = append(functionapps, *data.Name)
		}

		notDone = faList.NotDone()
		if notDone != true {
			break
		}
		faList.Next()
	}

	fmt.Println(functionapps)

	// Check if source control has been set.
	// for _, faName := range functionapps {
	// 	faSource, err := faClient.GetSourceControl(context.Background(), "mxi-dev", faName) // TODO: envs
	// 	if err != nil {
	// 		log.Println(err)
	// 	}
	// 	jsonData, _ := json.Marshal(faSource)
	// 	fmt.Println(string(jsonData))
	// }
}

func LogRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpRequestOut(r, true)
			fmt.Println("##### REQUEST #####")
			log.Println(string(dump))
			return r, err
		})
	}
}

func LogResponse() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			err := p.Respond(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println("##### RESPONSE #####")
			log.Println(string(dump))
			return err
		})
	}
}
