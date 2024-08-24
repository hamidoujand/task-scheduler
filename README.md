# Task Runner Service

The Task Runner Service is a robust and flexible backend system designed to manage and execute tasks within a containerized environment. This project is a portfolio piece, demonstrating my expertise in Go, Docker, and microservices architecture. It is not intended for production use.

## Description

The Task Runner Service enables users to manage and execute asynchronous tasks within Docker containers. It allows users to specify Docker images, execute commands within those images, and schedule tasks with precision. 

This project is built primarily using Go's standard library, minimizing dependencies to ensure simplicity and efficiency. It leverages Go's powerful concurrency features, such as goroutines and channels, to execute tasks concurrently, allowing for high performance and scalability in handling multiple tasks simultaneously.


## Features

- **Task Scheduling**: Schedule tasks to be executed at specific times.
- **Docker Command Execution**: Run commands inside Docker containers, using user-specified images.
- **Logging and Error Handling**: Detailed logs and error handling for each task execution.
- **User Management**: Manage users who can schedule and execute tasks.
- **JWT Authentication**: Secure user authentication using JWT tokens.
- **Microservice Architecture**: Designed with a microservice architecture for scalability and maintainability.

## Technologies Used

- **Go**: The core language used for developing the service.
- **Docker**: For containerization and task execution within isolated environments.
- **Docker Compose**: Facilitates local development and multi-container setups.
- **Redis**: Used for task queue management and caching.
- **RabbitMQ**: Message broker for handling task queues and communication between services.
- **PostgreSQL**: Relational database for storing user data and task metadata.
- **JWT Auth**: Secure user authentication with JSON Web Tokens (JWT).
- **Concurrency**: Leverages Go's powerful concurrency features to execute tasks concurrently, maximizing efficiency and performance.

## Installation

To set up the Task Runner Service locally, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/hamidoujand/task-scheduler.git
   cd task-scheduler

2. **Download Docker Images**: Use the provided Makefile to download the necessary Docker images
   ```bash
   make docker-pull

3. **Build the Docker Image**: Build the main service image using the provided Makefile
   ```bash
    make build

4. **Start the Services**: Use Docker Compose to start all services (PostgreSQL, Redis, RabbitMQ, and the Tasks service )
   ```bash
    make up

5. **Run Tests**: Run tests to ensure everything is set up correctly
   ```bash
    make test

6. **Stop the Services**: When you are done, you can stop all services
   ```bash
    make down

7. **Clean Up**: To remove temporary data and clean up your environment
   ```bash
    make clean

## Usage

Once the service is running, you can:

- Schedule tasks by specifying a Docker image and command.
- Manage users and assign them tasks.
- Monitor task execution through logs and handle any errors that arise.

## Hardcoded RSA Key for Development

For development purposes, this project uses a hardcoded RSA256 key located in the `zarf` directory. If you need to change this key, place a private key inside the `/zarf/keys/<key_id>.pem` directory.

## Logs

To view the service logs, you can use the following command:
```bash
make logs 
```

## Dependency Management
```bash
make tidy
```

## API Endpoints

The Task Runner Service provides the following API endpoints. Authentication with a JWT token is required for most endpoints, and some routes require specific user roles.

### Tasks Endpoints

- **Create Task**
  - **Method**: `POST`
  - **Path**: `/api/tasks/`
  - **Description**: Create a new task.
  - **Authentication**: Required (JWT)

- **Get Task by ID**
  - **Method**: `GET`
  - **Path**: `/api/tasks/{id}`
  - **Description**: Retrieve details of a task by its ID.
  - **Parameters**:
    - `{id}`: The ID of the task.
  - **Authentication**: Required (JWT)

- **Delete Task by ID**
  - **Method**: `DELETE`
  - **Path**: `/api/tasks/{id}`
  - **Description**: Delete a task by its ID.
  - **Parameters**:
    - `{id}`: The ID of the task.
  - **Authentication**: Required (JWT)

### Users Endpoints

- **Create User**
  - **Method**: `POST`
  - **Path**: `/api/users/`
  - **Description**: Create a new user.
  - **Authentication**: Required (JWT)
  - **Authorization**: Required (Role: Admin)

- **User Login**
  - **Method**: `POST`
  - **Path**: `/api/users/login`
  - **Description**: Log in as a user.
  - **Authentication**: Not required

- **User Signup**
  - **Method**: `POST`
  - **Path**: `/api/users/signup`
  - **Description**: Sign up a new user.
  - **Authentication**: Not required

- **Update User Role**
  - **Method**: `PUT`
  - **Path**: `/api/users/role/{id}`
  - **Description**: Update the role of a user.
  - **Parameters**:
    - `{id}`: The ID of the user.
  - **Authentication**: Required (JWT)
  - **Authorization**: Required (Role: Admin)

- **Get User by ID**
  - **Method**: `GET`
  - **Path**: `/api/users/{id}`
  - **Description**: Retrieve details of a user by their ID.
  - **Parameters**:
    - `{id}`: The ID of the user.
  - **Authentication**: Not required

- **Update User**
  - **Method**: `PUT`
  - **Path**: `/api/users/{id}`
  - **Description**: Update user details.
  - **Parameters**:
    - `{id}`: The ID of the user.
  - **Authentication**: Required (JWT)

- **Delete User by ID**
  - **Method**: `DELETE`
  - **Path**: `/api/users/{id}`
  - **Description**: Delete a user by their ID.
  - **Parameters**:
    - `{id}`: The ID of the user.
  - **Authentication**: Required (JWT)






