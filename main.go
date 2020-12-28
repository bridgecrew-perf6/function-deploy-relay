package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2020-06-01/web"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

var wg sync.WaitGroup

func main() {
	// Verify environment variables exist.
	// AZ_RG can contain multiple resource groups seperated by a comma.
	envs := CheckEnvs("GIT_URL", "AZ_RG", "AZ_SUB_ID", "AZURE_TENANT_ID", "AZURE_CLIENT_SECRET", "AZURE_CLIENT_ID")

	rg, subID, repoURL := envs["AZ_RG"], envs["AZ_SUB_ID"], envs["GIT_URL"]

	faClient := web.NewAppsClient(subID)
	// Auth.
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err == nil {
		faClient.Authorizer = authorizer
	}

	// faClient.RequestInspector = LogRequest()
	// faClient.ResponseInspector = LogResponse()

	// List function names for all resource groups, retreive publishing credentials, then trigger new deployments.
	x := regexp.MustCompile(`,`)
	rgs := x.Split(rg, -1)
	for i, rGroup := range rgs {
		fmt.Println("Function apps for resource group: ", "(", i+1, ")", rGroup, ":")
		azFuncs := ListFuncs(rGroup, faClient)
		// CheckSource(rGroup, azFuncs, faClient)
		creds := GetCreds(rGroup, azFuncs, faClient)

		// Create channel
		ch := make(chan string, len(creds))

		// Trigger deployment for each function.
		for key, cred := range creds {
			log.Println("Deploying to function: ", key)
			go Deploy(key, repoURL, cred, ch)
			wg.Add(1)
		}
		fmt.Println("\n")
		fmt.Println("\n")

		// Wait for replies and close the channel.
		wg.Wait()
		close(ch)

		// Print responses.
		for x := range ch {
			fmt.Println(x)
		}

	}
}

// CheckEnvs - verify envs are set.
func CheckEnvs(envs ...string) map[string]string {
	fmt.Println("Checking for environment variables:")
	envMap := make(map[string]string)
	for i, env := range envs {
		val, ok := os.LookupEnv(envs[i])
		if !ok {
			fmt.Printf("Environment variable: %s not set\n", env)
			os.Exit(1)
		} else {
			envMap[env] = val
			fmt.Printf("%s = %s\n", env, val)
		}
	}
	fmt.Println("\n")
	return envMap
}

// ListFuncs - List all Function Apps in resource group.
func ListFuncs(rg string, client web.AppsClient) []string {

	faList, err := client.ListComplete(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	// Create empty slice for function names.
	functionapps := []string{}

	// Set the regex to search for resource group name.
	matched := regexp.MustCompile(`\"serverFarmId\":"\/subscriptions\/.*\/resourceGroups\/` + rg)

	// Loop through all functions, adding matches into the slice.
	for notDone := true; notDone; notDone = faList.NotDone() {
		data := faList.Value()
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatal(err)
		}

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

	fmt.Println("\n")
	return functionapps

}

// GetCreds - publishing user for all functionapps.
func GetCreds(rg string, functionapps []string, client web.AppsClient) map[string]string {

	faMap := make(map[string]string)

	for _, faName := range functionapps {
		faUser, _ := client.ListPublishingCredentials(context.Background(), rg, faName)
		user, _ := faUser.Result(client)
		jsonUri, _ := json.Marshal(user.ScmURI)

		// Remove double quotes from URI and add it to the map.
		faMap[faName] = strings.Trim(string(jsonUri), "\"")
	}
	// fmt.Println(faMap, "\n")
	return faMap
}

// CheckSource - Check if source control has been set.
func CheckSource(name string, rg string, functionapps []string, client web.AppsClient) {

	for _, faName := range functionapps {
		faSource, err := client.GetSourceControl(context.Background(), rg, faName)
		if err != nil {
			log.Println(err)
		}
		jsonData, _ := json.Marshal(faSource)
		log.Println(faName, " : ", string(jsonData))
	}
	fmt.Println("\n")
}

// Deploy - Trigger deploy via the Azure Kudu service.
func Deploy(name, repoURL, publishURL string, ch chan<- string) {
	// Marshal JSON
	requestBody, err := json.Marshal(map[string]string{
		"format": "basic",
		"url":    repoURL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Sync
	resp := Post(name, publishURL, requestBody)

	// Retry sync
	for i := 1; i <= 3; i++ {
		if resp.StatusCode != 200 {
			fmt.Println(i, " - Retrying ", name, ": ", string(resp.Status), "\n")
			time.Sleep(5 * time.Second)
			resp = Post(name, publishURL, requestBody)
			if resp.StatusCode != 200 && i == 3 {
				log.Println("DEPLOY SYNC FAILED for ", name)
			}
		}
	}

	ch <- name + ":" + string(resp.Status) + "\n"
	wg.Done()
	return

}

// Send POST to Kudu.
func Post(name, publishURL string, requestBody []byte) *http.Response {
	// Send POST
	resp, err := http.Post(publishURL+"/deploy", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println(publishURL + "/deploy")
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	return resp
}

func LogRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				log.Println(err)
			}
			dump, _ := httputil.DumpRequestOut(r, true)
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
			log.Println(string(dump))
			return err
		})
	}
}
