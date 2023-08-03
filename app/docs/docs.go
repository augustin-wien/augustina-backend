// Code generated by swaggo/swag. DO NOT EDIT.

package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/vendors/": {
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
        "/api/vendors/{id}": {
            "put": {
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
        "/payments": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Get all payments",
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
                "description": "{\"Payments\":[{\"Sender\": 1, \"Receiver\":1, \"Type\":1,\"Amount\":1.00]}",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "core"
                ],
                "summary": "Create a set of payments",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/database.PaymentType"
                            }
                        }
                    }
                }
            }
        },
        "/settings/": {
            "get": {
                "description": "Return settings about the web-shop",
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
            }
        }
    },
    "definitions": {
        "database.Item": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "integer"
                },
                "image": {
                    "type": "string"
                },
                "isEditable": {
                    "type": "boolean"
                },
                "name": {
                    "type": "string"
                },
                "price": {
                    "type": "number"
                }
            }
        },
        "database.Payment": {
            "type": "object",
            "properties": {
                "amount": {
                    "type": "number"
                },
                "authorizedBy": {
                    "$ref": "#/definitions/pgtype.Int4"
                },
                "id": {
                    "type": "integer"
                },
                "item": {
                    "$ref": "#/definitions/pgtype.Int4"
                },
                "paymentBatch": {
                    "$ref": "#/definitions/pgtype.Int8"
                },
                "receiver": {
                    "type": "integer"
                },
                "sender": {
                    "type": "integer"
                },
                "timestamp": {
                    "$ref": "#/definitions/pgtype.Timestamp"
                },
                "type": {
                    "type": "integer"
                }
            }
        },
        "database.PaymentType": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "integer"
                },
                "name": {
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
                "newspaper": {
                    "$ref": "#/definitions/database.Item"
                },
                "refundFees": {
                    "type": "boolean"
                }
            }
        },
        "database.Vendor": {
            "type": "object",
            "properties": {
                "account": {
                    "type": "integer"
                },
                "balance": {
                    "description": "This is joined in from the account",
                    "type": "number"
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
                    "type": "string"
                },
                "licenseID": {
                    "type": "string"
                },
                "urlID": {
                    "description": "This is used for the QR code",
                    "type": "string"
                }
            }
        },
        "pgtype.InfinityModifier": {
            "type": "integer",
            "enum": [
                1,
                0,
                -1
            ],
            "x-enum-varnames": [
                "Infinity",
                "Finite",
                "NegativeInfinity"
            ]
        },
        "pgtype.Int4": {
            "type": "object",
            "properties": {
                "int32": {
                    "type": "integer"
                },
                "valid": {
                    "type": "boolean"
                }
            }
        },
        "pgtype.Int8": {
            "type": "object",
            "properties": {
                "int64": {
                    "type": "integer"
                },
                "valid": {
                    "type": "boolean"
                }
            }
        },
        "pgtype.Timestamp": {
            "type": "object",
            "properties": {
                "infinityModifier": {
                    "$ref": "#/definitions/pgtype.InfinityModifier"
                },
                "time": {
                    "description": "Time zone will be ignored when encoding to PostgreSQL.",
                    "type": "string"
                },
                "valid": {
                    "type": "boolean"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
