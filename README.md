# QuattriniTrack
***QuattriniTrack*** (from quattrini, meaning “money” in the Florentine dialect of Italian) is a personal finance manager designed to help you track expenses and manage your personal budget.
The app is built using the following technologies:
- The **Go** programming language. Using its rich standard library for all its features, especially to manage the underlying **REST API** of the program.
- The **SQLC** package to manage the database (in this case a local sqlite db). Fantastic way to write directly SQL queries and schemas and then translate it in a type safe manner to use in the program.
- The **Charm** ecosystem for the *TUI* (terminal user interface). The user is meant to use the program through the TUI, but the REST API remains available to be used through other means (such as curl).

## How to run
First of all make sure you have installed both the Go programming language and SQLC. Also the experience will be way smoother using a terminal with *ANSI* support. A step by step guide is the following:
- Clone the repository: ```git clone https://github.com/efive-dev/QuattriniTrack.git```.
- Navigate to the main directory where you cloned the repository.
- Create a *.env* file and in the first line insert ```JWT_SECRET = "your_secret_key"```, your_secret_key should be generated through a trusted service but any string will work.
- Run the program: ```go run .```
- Use the program through the TUI. The API will be available on the following address: ```http://localhost:8080/```.

## Database Schema
### Transactions:
The transactions table is the beating heart of the whole program. It models your expenses.
| Column          | Type     | Constraints                                |
| --------------- | -------- | ------------------------------------------ |
| `id`            | INTEGER  | Primary Key, Auto-increment                |
| `name`          | TEXT     | Not Null                                   |
| `cost`          | REAL     | Not Null, Must be > 0 (`CHECK (cost > 0)`) |
| `date`          | DATETIME | Not Null                                   |
| `categories_id` | INTEGER  | Not Null, Foreign Key → `categories(id)`   |

### Categories:
The categories table models user-defined classes of transactions.
| Column | Type    | Constraints                 |
| ------ | ------- | --------------------------- |
| `id`   | INTEGER | Primary Key, Auto-increment |
| `name` | TEXT    | Not Null, Unique            |

### Users:
The users table is used to store informations used for authentication and authorization.
| Column          | Type    | Constraints                 |
| --------------- | ------- | --------------------------- |
| `id`            | INTEGER | Primary Key, Auto-increment |
| `email`         | TEXT    | Not Null, Unique            |
| `password_hash` | TEXT    | Not Null                    |

## Endpoints
The API is organized into:
- Public routes: Available without authentication (e.g., user registration and login).
- Protected routes: Require a valid **JWT** to access (e.g., transaction and category management, user profile).

Authentication is handled using JWT (JSON Web Tokens). To access protected routes, clients must include a valid token in the Authorization header using the Bearer <token> format.

| Method | Path           | Description              | Auth Required |
| ------ | -------------- | ------------------------ | ------------- |
| POST   | `/register`    | Register a new user      | No            |
| POST   | `/login`       | Log in a user            | No            |
| GET    | `/transaction` | List all transactions    | Yes           |
| POST   | `/transaction` | Create a transaction     | Yes           |
| DELETE | `/transaction` | Delete a transaction     | Yes           |
| GET    | `/category`    | List categories          | Yes           |
| POST   | `/category`    | Create a category        | Yes           |
| DELETE | `/category`    | Delete a category        | Yes           |
| GET    | `/me`          | Get current user profile | Yes           |

Certain endpoints also allow filtering with query parameters:
- ```/transaction``` allows to filter based on id, categories_id and name.
- ```/category``` allows to filter based on id.

### Sample curl requests
1. Register a new user (public)
```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "your_password"}'
```
2. Log in and get JWT token (public)
```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "your_password"}'
```
3. Get all transactions (protected — requires JWT token)
```bash
curl -X GET http://localhost:8080/transaction \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE"
```
