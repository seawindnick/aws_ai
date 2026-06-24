# Shopping Cart - Serverless Demo App

A shopping cart application built on AWS serverless infrastructure, used in the Claude Code workshop to demonstrate agentic coding workflows.

## Architecture

```
┌─────────────┐        ┌──────────────┐        ┌───────────┐
│   React     │  HTTP  │ API Gateway  │  Proxy  │  Lambda   │
│  Frontend   │───────▶│   (REST)     │────────▶│  (Flask)  │
│  (S3)       │        │              │         │           │
└─────────────┘        └──────────────┘         └─────┬─────┘
                                                      │
                                                      ▼
                                                ┌───────────┐
                                                │ DynamoDB   │
                                                │ (3 tables) │
                                                └───────────┘
```

- **Frontend**: React (Vite) hosted on S3 with static website hosting
- **API Layer**: Amazon API Gateway (REST) with CORS enabled
- **Backend**: AWS Lambda running Flask via serverless-wsgi
- **Database**: DynamoDB (products, cart, orders tables)
- **IaC**: AWS SAM (Serverless Application Model)

## Project Structure

```
shopping-cart/
├── backend/
│   ├── app.py                  # Flask app entry point
│   ├── lambda_handler.py       # Lambda adapter (API Gateway → Flask)
│   ├── requirements.txt        # Python dependencies
│   ├── routes/
│   │   ├── products.py         # Product endpoints
│   │   ├── cart.py             # Cart endpoints
│   │   └── orders.py          # Order endpoints
│   ├── services/
│   │   ├── product_service.py  # Product business logic
│   │   ├── cart_service.py     # Cart business logic
│   │   └── order_service.py    # Order business logic
│   ├── models/
│   │   ├── product.py          # DynamoDB product model
│   │   ├── cart.py             # DynamoDB cart model
│   │   └── order.py            # DynamoDB order model
│   └── tests/
│       └── test_cart_service.py # Unit tests
├── frontend/
│   ├── index.html              # HTML entry point
│   ├── package.json            # Node dependencies
│   ├── vite.config.js          # Vite config (dev proxy)
│   └── src/
│       ├── main.jsx            # React entry point
│       ├── App.jsx             # Root component with routing
│       ├── styles.css          # Global styles
│       ├── components/
│       │   ├── Header.jsx      # Navigation header
│       │   ├── ProductCard.jsx # Product display card
│       │   └── CartItem.jsx    # Cart item with quantity controls
│       ├── pages/
│       │   ├── ProductList.jsx # Product listing with filters
│       │   ├── CartPage.jsx    # Cart view with checkout
│       │   └── OrdersPage.jsx  # Order history
│       ├── context/
│       │   └── CartContext.jsx # Cart state management
│       └── services/
│           └── api.js          # API client (all HTTP calls)
├── template.yaml               # SAM/CloudFormation template
└── deploy.sh                   # One-command deploy script
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /api/products | List products (optional: ?category=X) |
| GET | /api/products/:id | Get product details |
| GET | /api/products/categories | List categories |
| GET | /api/products/search?q=X | Search products |
| GET | /api/cart | Get cart with totals |
| POST | /api/cart/items | Add item to cart |
| PUT | /api/cart/items/:id | Update item quantity |
| DELETE | /api/cart/items/:id | Remove item |
| DELETE | /api/cart | Clear cart |
| POST | /api/orders | Place order (checkout) |
| GET | /api/orders | List order history |
| GET | /api/orders/:id | Get order details |

## Deploy

```bash
./deploy.sh
```

This single command:
1. Builds the Lambda package (SAM build)
2. Deploys backend infrastructure (API Gateway + Lambda + DynamoDB)
3. Builds the React frontend (with the API URL injected)
4. Uploads the frontend to S3

## Local Development

```bash
# Backend
cd backend
pip install -r requirements.txt
python app.py

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

The Vite dev server proxies `/api` requests to `localhost:5000`.

## Redeploy After Changes

After Claude makes code changes, run:

```bash
./deploy.sh
```

Refresh the browser to see changes live.
