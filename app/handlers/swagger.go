//	@title			Augustin Swagger
//	@version		0.0.1
//	@description	This swagger describes every endpoint of this project.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	GNU Affero General Public License
//	@license.url	https://www.gnu.org/licenses/agpl-3.0.txt

//	@host		localhost:3000
//	@BasePath	/api
// @accept json

//	@securityDefinitions.basic	BasicAuth

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/

// @securityDefinitions.apikey KeycloakAuth
// @in header
// @name Authorization
// @description	<b>how to generate an api key</b> <br/><br/><code>curl -d 'client_id=frontend' -d 'scope=openid' -d 'username=test_superuser' -d 'password=Test123!' -d 'grant_type=password' 'http://keycloak:8080/realms/augustin/protocol/openid-connect/token' |     python3 -m json.tool | grep access_token</code><br/><br/><br/>Insert the output into the field below the value of the access_token field.

// Package handlers contains all the handlers for the API
package handlers
