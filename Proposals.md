# Proposals

## Agent Actions

* Subscribe to operations
* Polling or Subscriptions
* Update Agent Configuration (i.e. the c8y_Configuration)
* 

### Process operations

* Listen for set of given operations? Only call back on matching operations [Only relavent for polling]
* Set status of Operations to PENDING
* Perform an operation (sychronously)
* Set status of Operation to FAILED or SUCCESSFUL *If set to FAILED, then set a REASON
* 

#### Requirements

* Use workers (channels)
* 

## TODO

* /user
* /tenant
* /retentionRule
* /application/201/binaries
* /event/events/binaries
* GET /measurement/measurements/<<measurementId>>
* DELETE /measurement/measurements/<<measurementID>>
* DELETE: /measurement/measurements (delete measurement collection)
* POST /measurement/measurements (post multiple measurements)
* Measurements: Allow setting header 'X-Cumulocity-System-Of-Units' to either "imperial" or "metric" (global setting?)
* 
