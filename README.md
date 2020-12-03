# Azure Deployment Relay for Functions
Azure Functions (and App Services) can be deployed automatically via commits to a git repository, providing for a simple and free [continuous delivery](https://docs.microsoft.com/en-us/azure/app-service/deploy-continuous-deployment) process. Setting this up for a Function creates a unique deploy key and a webhook in the Github repo. Github has a limit of 20 webhooks per repo so this tool was created to act as a relay between the Github webhook events and the Kudu deployment service, allowing any number of Functions to have deployments triggered from a single repo. 

## Tools used:
- [Kudu](https://github.com/projectkudu/kudu/wiki/Manually-triggering-a-deployment)
- [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go)


## Required Environment Variables
- AZ_RG - Resource group the Azure FunctionApp resides in. Mutliple groups can be specified with a comma.
- AZ_SUB_ID - Azure Subscription ID.
- GIT_URL - URL of the repository. This has only been tested with the SSH type URLs. HTTPS should work as long as the https://username:password@domain/path/repo.git format is used.

## Variables required for authentication to Azure API.
- AZURE_TENANT_ID
- AZURE_CLIENT_ID
- AZURE_CLIENT_SECRET

