// Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "Blink Labs",
            "url": "https://blinklabs.io",
            "email": "support@blinklabs.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/localtxmonitor/has_tx/{tx_hash}": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "localtxmonitor"
                ],
                "summary": "Check if a particular TX exists in the mempool",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.responseLocalTxMonitorHasTx"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.responseApiError"
                        }
                    }
                }
            }
        },
        "/localtxmonitor/sizes": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "localtxmonitor"
                ],
                "summary": "Get mempool capacity, size, and TX count",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.responseLocalTxMonitorSizes"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.responseApiError"
                        }
                    }
                }
            }
        },
        "/localtxmonitor/txs": {
            "get": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "localtxmonitor"
                ],
                "summary": "List all transactions in the mempool",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/api.responseLocalTxMonitorTxs"
                            }
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.responseApiError"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "api.responseApiError": {
            "type": "object",
            "properties": {
                "msg": {
                    "type": "string",
                    "example": "error message"
                }
            }
        },
        "api.responseLocalTxMonitorHasTx": {
            "type": "object",
            "properties": {
                "has_tx": {
                    "type": "boolean"
                }
            }
        },
        "api.responseLocalTxMonitorSizes": {
            "type": "object",
            "properties": {
                "capacity": {
                    "type": "integer"
                },
                "size": {
                    "type": "integer"
                },
                "tx_count": {
                    "type": "integer"
                }
            }
        },
        "api.responseLocalTxMonitorTxs": {
            "type": "object",
            "properties": {
                "tx_bytes": {
                    "type": "string",
                    "format": "base64",
                    "example": "\u003cbase64 encoded transaction bytes\u003e"
                },
                "tx_hash": {
                    "type": "string",
                    "format": "base16",
                    "example": "96649a8b827a5a4d508cd4e98cd88832482f7b884d507a49466d1fb8c4b14978"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost",
	BasePath:         "/api",
	Schemes:          []string{"http"},
	Title:            "cardano-node-api",
	Description:      "Cardano Node API",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
