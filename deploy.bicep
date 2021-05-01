param containerName string = 'spellapi'
param containerVersion string = 'main'

param ImageRegistry string = 'halbarad.azurecr.io'
param ImageRegistryUsername string = 'halbarad'

@secure()
param ImageRegistryPassword string
param HoneycombDataset string

@secure()
param HoneycombApiKey string


resource bot_aci 'Microsoft.ContainerInstance/containerGroups@2018-10-01' = {
  name: containerName
  location: 'westeurope'
  properties: {
    containers: [
      {
        name: containerName
        properties: {
          image: '${ImageRegistry}/go/${containerName}:${containerVersion}'
          ports: [
            {
              protocol: 'TCP'
              port: 80
            }
          ]
          environmentVariables: [
            {
              name: 'HONEYCOMB_KEY'
              value: HoneycombApiKey
            }
            {
              name: 'HONEYCOMB_DATASET'
              value: HoneycombDataset
            }
            {
              name: 'COSMOSDB_URI'
              value: listConnectionStrings(cosmosdb.id, '2020-04-01').connectionStrings[0].connectionString
            }
          ]
          resources: {
            requests: {
              memoryInGB: '1.5'
              cpu: 1
            }
          }
        }
      }
    ]
    imageRegistryCredentials: [
      {
        server: ImageRegistry
        username: ImageRegistryUsername
        password: ImageRegistryPassword
      }
    ]
    restartPolicy: 'OnFailure'
    ipAddress: {
      ports: [
        {
          protocol: 'TCP'
          port: 80
        }
      ]
      type: 'Public'
      dnsNameLabel: containerName
    }
    osType: 'Linux'
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
