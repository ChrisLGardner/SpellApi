param containerName string = 'spellapi'
param containerVersion string = 'main'

param ImageRegistry string = 'halbarad.azurecr.io'
param ImageRegistryUsername string = 'halbarad'

@secure()
param ImageRegistryPassword string
param HoneycombDataset string

@secure()
param HoneycombApiKey string

param website_name string = 'spellapi'

resource website_name_web 'Microsoft.Web/sites/config@2018-11-01' = {
  name: '${website_name}/web'
  properties: {
    linuxFxVersion: 'DOCKER|${ImageRegistry}/go/${containerName}:${containerVersion}'
  }
}

resource website_name_appsettings 'Microsoft.Web/sites/config@2018-11-01' = {
  name: '${website_name}/appsettings'
  properties: {
    COSMOSDB_URI: listConnectionStrings(cosmosdb.id, '2020-04-01').connectionStrings[0].connectionString
    DOCKER_REGISTRY_SERVER_PASSWORD: ImageRegistryPassword
    DOCKER_REGISTRY_SERVER_URL: 'https://${ImageRegistry}'
    DOCKER_REGISTRY_SERVER_USERNAME: ImageRegistryUsername
    HONEYCOMB_DATASET: HoneycombDataset
    HONEYCOMB_KEY: HoneycombApiKey
    WEBSITES_ENABLE_APP_SERVICE_STORAGE: false
    PORT: 443
  }
}

resource cosmosdb 'Microsoft.DocumentDb/databaseAccounts@2020-04-01' = {
  kind: 'MongoDB'
  name: containerName
  location: 'westeurope'
  properties: {
    databaseAccountOfferType: 'Standard'
    locations: [
      {
        id: '${containerName}-westeurope'
        failoverPriority: 0
        locationName: 'westeurope'
      }
    ]
    backupPolicy: {
      type: 'Periodic'
      periodicModeProperties: {
        backupIntervalInMinutes: 1440
        backupRetentionIntervalInHours: 48
      }
    }
    isVirtualNetworkFilterEnabled: false
    virtualNetworkRules: []
    ipRules: []
    dependsOn: []
    enableMultipleWriteLocations: false
    capabilities: [
      {
        name: 'EnableMongo'
      }
      {
        name: 'DisableRateLimitingResponses'
      }
    ]
    apiProperties: {
      serverVersion: '3.6'
    }
  }
  tags: {
    defaultExperience: 'Azure Cosmos DB for MongoDB API'
    'hidden-cosmos-mmspecial': ''
    CosmosAccountType: 'Non-Production'
  }
}
