# SpellApi
The goal of this project is to provide a generic API for managing spells for various TTRPGs. Originally I set out to create a place for me to store spells created by players in a Mage: The Awakening game I was planning on running and set up a Discord bot to allow easier retrieval and searching while playing/planning. This led to wanting to make it a bit more generic and flexible for use with other systems and by anyone else.

This is an API first to allow more options for integration and future work will be done to provide a web UI and a basic Discord bot.

If there are any features that people would like to see then please create a new issue with as much detail as possible or submit a PR (or both).

## API Defintion

This is a basic overview of the API and I'll aim to keep this up to date as I work on this more. The docs will also be available from the root of the API eventually.

### GET /spells/{name}

Returns a specific spell, if there are multiple with the same name then you can add filters using URL query parameters to narrow it down. A common parameter to use is `system`

#### Examples
```
Request:

GET /spells/cure%20wounds

Response:

{
    "name":"Cure Wounds",
    "description":"Heals a target for 10HP",
    "metadata":{
        "system":"Random System 1"
    }
}


Request:

GET /spells/fireball?system=test1

Response:

{
    "name":"Fireball",
    "description":"Deals 3 levels of Fire damage to all enemies within 10m of the target point.",
    "metadata":{
        "system":"test1"
    }
}
```

### POST /spells

Create a spell while specifying some useful metadata to make it more searchable etc. Only 1 spell of a given name can exist for each system. Modifying existing spells will be available via the PUT and/or PATCH methods on /spells/{name} `coming soon`.

```
Request:

POST /spells

{
    "name": "fireball",
    "description": "Deals 3 levels of Fire damage to all enemies within 10m of the target point.",
    "spelldata":{
        "level": 2,
        "school": "evocation",
        "type": "fire"
    }
    "metadata":{
        "system":"test1"
    }
}

Resonse:

201 Created
```


### Spell defintion

All of the `/spells` endpoints either accept or return objects of the [Spell](spell.go) type which has the following properties and requirements.

|Property|Required?|Description|
|---|---|---|
|name|Yes|Name of the spell. Can include any alphanumeric characters and spaces. This will be the primary way users search for spells.|
|description|Yes|Description of the spell. Can include any alphanumeric characters and spaces. Should be the relevant game text for what the spell does, anything around casting time etc should go in `spelldata`.|
|spelldata|No|System-specific information about the spell such as casting time, level etc. Accepts a map of key:value pairs.|
|metadata|Yes|Non-spell information about the spell. Detailed below under [SpellMetadata Definition](#spellmetadatadefintion).|

### SpellMetadata Definition

The `metadata` property of the `Spell` type stores a few pieces of general information about the spell that aren't directly related to gameplay. Currently this includes the following properties and requirements.

|Property|Required?|Description|
|---|---|---|
|system|Yes|Name of the game system the spell is for. This is used to ensure there is only a single spell of a particular name per system.|
|creator|No|Name/ID of the creator of the spell. Not currently used for anything but may be in future for updating spells.|
