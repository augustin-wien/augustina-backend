// Code generated by swaggo/swag. DO NOT EDIT.

package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "consumes": [
        "application/json"
    ],
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "GNU Affero General Public License",
            "url": "https://www.gnu.org/licenses/agpl-3.0.txt"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/hello/": {
            "get": {
                "description": "Return HelloWorld as sample API call",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Return HelloWorld",
                "responses": {}
            }
        },
        "/items/": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Items"
                ],
                "summary": "List Items",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/database.Item"
                            }
                        }
                    }
                }
            },
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Items"
                ],
                "summary": "Create Item",
                "parameters": [
                    {
                        "description": "Item Representation",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/database.Item"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "int"
                        }
                    }
                }
            }
        },
        "/items/{id}": {
            "delete": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Items"
                ],
                "summary": "Delete Item",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Item ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/items/{id}/": {
            "put": {
                "description": "Requires multipart form (for image)",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Items"
                ],
                "summary": "Update Item",
                "parameters": [
                    {
                        "description": "Item Representation",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/database.Item"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/orders/": {
            "post": {
                "description": "Submits payment order to provider \u0026 saves it to database. Entries need to have an item id and a quantity (for entries without a price like tips, the quantity is the amount of cents). If no user is given, the order is anonymous.",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Orders"
                ],
                "summary": "Create Payment Order",
                "parameters": [
                    {
                        "description": "Payment Order",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handlers.createOrderRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/handlers.createOrderResponse"
                        }
                    }
                }
            }
        },
        "/payments/": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Payments"
                ],
                "summary": "Get list of all payments",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/database.Payment"
                            }
                        }
                    }
                }
            },
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Payments"
                ],
                "summary": "Create a payment",
                "parameters": [
                    {
                        "description": " Create Payment",
                        "name": "amount",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/database.Payment"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/payments/batch/": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Payments"
                ],
                "summary": "Create a set of payments",
                "parameters": [
                    {
                        "description": " Create Payment",
                        "name": "amount",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handlers.createPaymentsRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "int"
                        }
                    }
                }
            }
        },
        "/payments/payout/": {
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Payments"
                ],
                "summary": "Create a payment from a vendor account to cash",
                "parameters": [
                    {
                        "description": " Create Payment",
                        "name": "amount",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/handlers.createPaymentPayoutRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "int"
                        }
                    }
                }
            }
        },
        "/settings/": {
            "get": {
                "description": "Return configuration data of the system",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Return settings",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/database.Settings"
                            }
                        }
                    }
                }
            },
            "put": {
                "description": "Update configuration data of the system",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Update settings",
                "parameters": [
                    {
                        "description": "Settings Representation",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/database.Settings"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/vendors/": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "vendors"
                ],
                "summary": "List Vendors",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/database.Vendor"
                            }
                        }
                    }
                }
            },
            "post": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "vendors"
                ],
                "summary": "Create Vendor",
                "parameters": [
                    {
                        "description": "Vendor Representation",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/database.Vendor"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/vendors/{id}/": {
            "put": {
                "description": "Warning: Unfilled fields will be set to default values",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "vendors"
                ],
                "summary": "Update Vendor",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Vendor ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Vendor Representation",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/database.Vendor"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            },
            "delete": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "vendors"
                ],
                "summary": "Delete Vendor",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Vendor ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/webhooks/vivawallet/failure": {
            "get": {
                "description": "Return VivaWallet verification key",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Return VivaWallet verification key",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/paymentprovider.VivaWalletVerificationKeyResponse"
                            }
                        }
                    }
                }
            },
            "post": {
                "description": "Webhook for VivaWallet failed transaction",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Webhook for VivaWallet failed transaction",
                "parameters": [
                    {
                        "description": "Payment Failure Response",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/paymentprovider.TransactionDetailRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/webhooks/vivawallet/price": {
            "get": {
                "description": "Return VivaWallet verification key",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Return VivaWallet verification key",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/paymentprovider.VivaWalletVerificationKeyResponse"
                            }
                        }
                    }
                }
            },
            "post": {
                "description": "Webhook for VivaWallet transaction prices",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Webhook for VivaWallet transaction prices",
                "parameters": [
                    {
                        "description": "Payment Price Response",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/paymentprovider.TransactionPriceRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        },
        "/webhooks/vivawallet/success": {
            "get": {
                "description": "Return VivaWallet verification key",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Return VivaWallet verification key",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/paymentprovider.VivaWalletVerificationKeyResponse"
                            }
                        }
                    }
                }
            },
            "post": {
                "description": "Webhook for VivaWallet successful transaction",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Webhook for VivaWallet successful transaction",
                "parameters": [
                    {
                        "description": "Payment Successful Response",
                        "name": "data",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/paymentprovider.TransactionDetailRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK"
                    }
                }
            }
        }
    },
    "definitions": {
        "database.Item": {
            "type": "object",
            "properties": {
                "archived": {
                    "type": "boolean"
                },
                "description": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "image": {
                    "type": "string"
                },
                "licenseItem": {
                    "description": "License has to be bought before item",
                    "allOf": [
                        {
                            "$ref": "#/definitions/null.Int"
                        }
                    ]
                },
                "name": {
                    "type": "string"
                },
                "price": {
                    "description": "Price in cents",
                    "type": "integer"
                }
            }
        },
        "database.Payment": {
            "type": "object",
            "properties": {
                "amount": {
                    "type": "integer"
                },
                "authorizedBy": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "order": {
                    "type": "integer"
                },
                "orderEntry": {
                    "type": "integer"
                },
                "receiver": {
                    "type": "integer"
                },
                "sender": {
                    "type": "integer"
                },
                "timestamp": {
                    "type": "string"
                }
            }
        },
        "database.Settings": {
            "type": "object",
            "properties": {
                "color": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "logo": {
                    "type": "string"
                },
                "mainItem": {
                    "type": "integer"
                },
                "refundFees": {
                    "type": "boolean"
                }
            }
        },
        "database.Vendor": {
            "type": "object",
            "properties": {
                "balance": {
                    "description": "This is joined in from the account",
                    "type": "integer"
                },
                "email": {
                    "type": "string"
                },
                "firstName": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "keycloakID": {
                    "type": "string"
                },
                "lastName": {
                    "type": "string"
                },
                "lastPayout": {
                    "type": "string",
                    "format": "date-time"
                },
                "licenseID": {
                    "type": "string"
                },
                "urlid": {
                    "description": "This is used for the QR code",
                    "type": "string"
                }
            }
        },
        "handlers.createOrderRequest": {
            "type": "object",
            "properties": {
                "entries": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/handlers.createOrderRequestEntry"
                    }
                },
                "user": {
                    "type": "string"
                },
                "vendor": {
                    "type": "integer"
                }
            }
        },
        "handlers.createOrderRequestEntry": {
            "type": "object",
            "properties": {
                "item": {
                    "type": "integer"
                },
                "quantity": {
                    "type": "integer"
                }
            }
        },
        "handlers.createOrderResponse": {
            "type": "object",
            "properties": {
                "smartCheckoutURL": {
                    "type": "string"
                }
            }
        },
        "handlers.createPaymentPayoutRequest": {
            "type": "object",
            "properties": {
                "amount": {
                    "type": "integer"
                },
                "vendorLicenseID": {
                    "type": "string"
                }
            }
        },
        "handlers.createPaymentsRequest": {
            "type": "object",
            "properties": {
                "payments": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/database.Payment"
                    }
                }
            }
        },
        "null.Int": {
            "type": "object",
            "properties": {
                "int64": {
                    "type": "integer"
                },
                "valid": {
                    "description": "Valid is true if Int64 is not NULL",
                    "type": "boolean"
                }
            }
        },
        "paymentprovider.EventData": {
            "type": "object",
            "properties": {
                "AcquirerApproved": {
                    "type": "boolean"
                },
                "Amount": {
                    "type": "number"
                },
                "AssignedMerchantUsers": {
                    "type": "array",
                    "items": {}
                },
                "AssignedResellerUsers": {
                    "type": "array",
                    "items": {}
                },
                "AuthorizationId": {
                    "type": "string"
                },
                "BankId": {
                    "type": "string"
                },
                "BillId": {},
                "BinId": {
                    "type": "integer"
                },
                "CardCountryCode": {},
                "CardExpirationDate": {
                    "type": "string"
                },
                "CardIssuingBank": {},
                "CardNumber": {
                    "type": "string"
                },
                "CardToken": {
                    "type": "string"
                },
                "CardTypeId": {
                    "type": "integer"
                },
                "CardUniqueReference": {
                    "type": "string"
                },
                "ChannelId": {
                    "type": "string"
                },
                "ClearanceDate": {},
                "CompanyName": {},
                "CompanyTitle": {},
                "ConnectedAccountId": {},
                "CurrencyCode": {
                    "type": "string"
                },
                "CurrentInstallment": {
                    "type": "integer"
                },
                "CustomerTrns": {
                    "type": "string"
                },
                "DigitalWalletId": {},
                "DualMessage": {
                    "type": "boolean"
                },
                "ElectronicCommerceIndicator": {
                    "type": "string"
                },
                "Email": {
                    "type": "string"
                },
                "FullName": {
                    "type": "string"
                },
                "InsDate": {
                    "type": "string"
                },
                "IsManualRefund": {
                    "type": "boolean"
                },
                "Latitude": {},
                "Longitude": {},
                "LoyaltyTriggered": {
                    "type": "boolean"
                },
                "MerchantCategoryCode": {
                    "type": "integer"
                },
                "MerchantId": {
                    "type": "string"
                },
                "MerchantTrns": {
                    "type": "string"
                },
                "Moto": {
                    "type": "boolean"
                },
                "OrderCode": {
                    "type": "integer"
                },
                "OrderCulture": {
                    "type": "string"
                },
                "OrderServiceId": {
                    "type": "integer"
                },
                "PanEntryMode": {
                    "type": "string"
                },
                "ParentId": {},
                "Phone": {
                    "type": "string"
                },
                "ProductId": {},
                "RedeemedAmount": {
                    "type": "number"
                },
                "ReferenceNumber": {
                    "type": "integer"
                },
                "ResellerCompanyName": {},
                "ResellerId": {},
                "ResellerSourceAddress": {},
                "ResellerSourceCode": {},
                "ResellerSourceName": {},
                "ResponseCode": {
                    "type": "string"
                },
                "ResponseEventId": {},
                "RetrievalReferenceNumber": {
                    "type": "string"
                },
                "ServiceId": {},
                "SourceCode": {
                    "type": "string"
                },
                "SourceName": {
                    "type": "string"
                },
                "StatusId": {
                    "type": "string"
                },
                "Switching": {
                    "type": "boolean"
                },
                "Systemic": {
                    "type": "boolean"
                },
                "Tags": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    }
                },
                "TargetPersonId": {},
                "TargetWalletId": {},
                "TerminalId": {
                    "type": "integer"
                },
                "TipAmount": {
                    "type": "number"
                },
                "TotalFee": {
                    "type": "number"
                },
                "TotalInstallments": {
                    "type": "integer"
                },
                "TransactionId": {
                    "type": "string"
                },
                "TransactionTypeId": {
                    "type": "integer"
                },
                "Ucaf": {
                    "type": "string"
                }
            }
        },
        "paymentprovider.PriceEventData": {
            "type": "object",
            "properties": {
                "CurrencyCode": {
                    "type": "string"
                },
                "Interchange": {
                    "type": "number"
                },
                "IsvFee": {
                    "type": "number"
                },
                "MerchantId": {
                    "type": "string"
                },
                "OrderCode": {
                    "type": "integer"
                },
                "TotalCommission": {
                    "type": "number"
                },
                "TransactionId": {
                    "type": "string"
                }
            }
        },
        "paymentprovider.TransactionDetailRequest": {
            "type": "object",
            "properties": {
                "CorrelationId": {
                    "type": "string"
                },
                "Created": {
                    "type": "string"
                },
                "Delay": {},
                "EventData": {
                    "$ref": "#/definitions/paymentprovider.EventData"
                },
                "EventTypeId": {
                    "type": "integer"
                },
                "MessageId": {
                    "type": "string"
                },
                "MessageTypeId": {
                    "type": "integer"
                },
                "RecipientId": {
                    "type": "string"
                },
                "Url": {
                    "type": "string"
                }
            }
        },
        "paymentprovider.TransactionPriceRequest": {
            "type": "object",
            "properties": {
                "CorrelationId": {
                    "type": "string"
                },
                "Created": {
                    "type": "string"
                },
                "Delay": {
                    "type": "integer"
                },
                "EventData": {
                    "$ref": "#/definitions/paymentprovider.PriceEventData"
                },
                "EventTypeId": {
                    "type": "integer"
                },
                "MessageId": {
                    "type": "string"
                },
                "MessageTypeId": {
                    "type": "integer"
                },
                "RecipientId": {
                    "type": "string"
                },
                "Url": {
                    "type": "string"
                }
            }
        },
        "paymentprovider.VivaWalletVerificationKeyResponse": {
            "type": "object",
            "properties": {
                "key": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "BasicAuth": {
            "type": "basic"
        }
    },
    "externalDocs": {
        "description": "OpenAPI",
        "url": "https://swagger.io/resources/open-api/"
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "0.0.1",
	Host:             "localhost:3000",
	BasePath:         "/api",
	Schemes:          []string{},
	Title:            "Augustin Swagger",
	Description:      "This swagger describes every endpoint of this project.",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
