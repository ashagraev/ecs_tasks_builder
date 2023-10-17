# ecs_tasks_builder

A simple program for generating json definitions of ECS tasks.

GitLab example for usage in CI/CD scripts:

```YAML
- aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $CONTAINER_REGISTRY
- docker build --tag=$IMAGE_TAG .
- docker push $IMAGE_TAG
- ecs_tasks_builder --source $ECS_TASK_FILENAME --output $ECS_TASK_FILENAME
  --dd_tag cloud:aws
  --dd_tag environment:production
  --dd_tag instance_type:fargate
  --env GUNICORN_WORKERS=$GUNICORN_WORKERS
  --container $SERVICE
  --tag $CI_COMMIT_SHORT_SHA
- aws ecs register-task-definition --region $REGION --cli-input-json file://$ECS_TASK_FILENAME
- aws ecs update-service --cluster $CLUSTER --service $SERVICE --task-definition $SERVICE --region $REGION
- aws ecs wait services-stable --cluster $CLUSTER --services $SERVICE --region $REGION
```

For GitHub actions, use `aws-actions/amazon-ecs-render-task-definition@97587c9d45a4930bf0e3da8dd2feb2a463cf4a3a` and
similar actions
instead.

The only non-trivial thing you'll need in advance is the task execution role. Create it [as noted in documentation](
https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_execution_IAM_role.html#create-task-execution-role) and
then use it in the `executionRoleArn` key in `ecs_task_definition_sample.json`.

Also, you'll find in this repo:

- sample GitHub action YML for CI/CD with AWS Fargate
- sample GitLab CI YML for CI/CD with AWS Fargate
- sample ECS Task Definition JSON for AWS Fargate tasks

In all cases, read the files thoroughly and change all the stubs with your data (like service code paths, service names,
regions, etc.).