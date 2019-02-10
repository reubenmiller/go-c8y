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
