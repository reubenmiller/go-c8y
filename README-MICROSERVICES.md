/*
Use cases:

Basic client:
1. Use credentials given by user (tenant, username, password)

Microservice
PER_TENANT

1. Given bootstrap credentials (from environment or specific creds), get the related service user
  - Save bootstrap credentials on the client
  - Save the service user credentials on the client
  - Use service user for all following requests

  * Have a default which uses the first tenant service user (also set from C8Y_BOOTSTRAP)

  * Option to override the bootstrap user with a service user.
  SetCredentialFromDefaultServiceUser()		<= Requires bootstrap user to be set
	- Get

  NewServiceUserContextFromEnvironment() ctx
  DefaultServiceUser()
  ServiceUser("")
  BootstrapUser()



MULTI_TENANT
1. Given bootstrap credentials (from env or specific credentials).
2. Allow requests to use service credentials (based on tenant), or bootstrap user or


Process for PER_TENANT
1. Read bootstrap credentials from Environment
2. Request application name (/currentApplication) and all service users
3. Set service user as default rest request user

* Allow the user to still use the bootstrap context, by using NewBootstrapUserContext()


Process for MULTI_TENANT
1. Read bootstrap credentials from Environment
2. Request application name (/currentApplication) and all service users
3. On each request set the context based on a tenant, allow user to loop through service users.
  -> cache service users, or request them each time (in case new subscriptions?)
*/
