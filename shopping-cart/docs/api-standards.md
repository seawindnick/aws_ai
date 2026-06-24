# API Coding Standards

## Endpoint Naming
- Use plural nouns for resources: `/api/products`, `/api/orders`
- Use nested routes for relationships: `/api/cart/items/{productId}`
- Use kebab-case for multi-word paths: `/api/order-history`

## HTTP Methods
- GET: Retrieve data (never modify state)
- POST: Create new resources
- PUT: Update entire resource
- DELETE: Remove a resource

## Request Validation
- Validate all input before processing
- Return 400 with `{"error": "description"}` for invalid input
- Check required fields explicitly — do not rely on database constraints

## Response Format
- Success: `{"data": ...}` or direct JSON object
- Error: `{"error": "human-readable message"}`
- List responses: `{"items": [...], "count": N}`
- Always return appropriate HTTP status codes:
  - 200: Success
  - 201: Created
  - 400: Bad request
  - 404: Not found
  - 500: Internal server error

## Error Handling
- Catch exceptions at the route level
- Never expose stack traces to the client
- Log errors with context (method, path, error message)
- Return generic 500 error for unexpected failures

## Data Access
- Never import boto3 or DynamoDB directly in route handlers
- Always go through the service layer for business logic
- Always go through the model layer for database access
- Keep route handlers thin — validate input, call service, return response
