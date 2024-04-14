resource "aws_ecr_repository" "app_ecr_repo" {
  name = "api-gateway"
}

resource "aws_ecs_cluster" "api_gateway_cluster" {
  name = "api-gateway-cluster"
}

resource "aws_cloudwatch_log_group" "api-gateway-server-group" {
  name              = "/ecs/api-gateway-server"
  retention_in_days = 30
}

resource "aws_ecs_task_definition" "api_gateway_server_task" {
  family                   = "api-gateway-server"
  container_definitions    = <<DEFINITION
  [
    {
      "name": "api-gateway-server",
      "image": "${aws_ecr_repository.app_ecr_repo.repository_url}",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
            "awslogs-group": "${aws_cloudwatch_log_group.api-gateway-server-group.name}",
            "awslogs-region": "${var.aws_region}",
            "awslogs-stream-prefix": "api-gateway-server-log-stream"
        }
      },
      "memory": 512,
      "cpu": 256
    }
  ]
  DEFINITION
  requires_compatibilities = ["FARGATE"] # use Fargate as the launch type
  network_mode             = "awsvpc"    # add the AWS VPN network mode as this is required for Fargate
  memory                   = 512         # Specify the memory the container requires
  cpu                      = 256         # Specify the CPU the container requires
  execution_role_arn       = aws_iam_role.ecsTaskExecutionRole.arn
}

resource "aws_iam_role" "ecsTaskExecutionRole" {
  name               = "ecsTaskExecutionRole"
  assume_role_policy = data.aws_iam_policy_document.assume_role_policy.json
}

data "aws_iam_policy_document" "assume_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "ecsTaskExecutionRole_policy" {
  role       = aws_iam_role.ecsTaskExecutionRole.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_ecs_service" "api-gateway-server-service" {
  name            = "api-gateway-server-service"
  cluster         = aws_ecs_cluster.api_gateway_cluster.id
  task_definition = aws_ecs_task_definition.api_gateway_server_task.arn
  launch_type     = "FARGATE"
  desired_count   = 3

  load_balancer {
    target_group_arn = aws_lb_target_group.target_group.arn
    container_name   = aws_ecs_task_definition.api_gateway_server_task.family
    container_port   = 8080
  }

  network_configuration {
    subnets          = ["${aws_subnet.private_subnet.id}"]
    assign_public_ip = false
    security_groups  = ["${aws_security_group.service_security_group.id}"]
  }
}

resource "aws_security_group" "service_security_group" {
  vpc_id = aws_vpc.app_vpc.id
  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    # Only allowing traffic in from the load balancer security group
    security_groups = ["${aws_security_group.load_balancer_security_group.id}"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
