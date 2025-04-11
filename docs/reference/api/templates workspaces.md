# Templates Workspaces

## Open dynamic parameters WebSocket by template version

### Code samples

```shell
# Example request using curl
curl -X GET http://coder-server:8080/api/v2/users/{user}/templateversion/{templateversion}/parameters \
  -H 'Coder-Session-Token: API_KEY'
```

`GET /users/{user}/templateversion/{templateversion}/parameters`

### Parameters

| Name              | In   | Type         | Required | Description         |
|-------------------|------|--------------|----------|---------------------|
| `user`            | path | string(uuid) | true     | Template version ID |
| `templateversion` | path | string(uuid) | true     | Template version ID |

### Responses

| Status | Meaning                                                                  | Description         | Schema |
|--------|--------------------------------------------------------------------------|---------------------|--------|
| 101    | [Switching Protocols](https://tools.ietf.org/html/rfc7231#section-6.2.2) | Switching Protocols |        |

To perform this operation, you must be authenticated. [Learn more](authentication.md).
