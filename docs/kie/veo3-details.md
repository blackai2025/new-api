# Get Veo3.1 Video Details

> Query the execution status and results of Veo3.1 video generation tasks.

<Info>
  Query the execution status and results of Veo3.1 video generation tasks.
</Info>

### Status Descriptions

- 0: Generating - Task is currently being processed
- 1: Success - Task completed successfully
- 2: Failed - Task generation failed
- 3: Generation Failed - Task created successfully but generation failed

### Important Notes

- Query task status in real-time using taskId
- Recommend periodic polling until task completion
- Response includes `fallbackFlag` field indicating whether fallback model was used

### Response Field Description

<ParamField path="fallbackFlag" type="boolean">
  Whether generated using fallback model. `true` means backup model was used, `false` means primary model was used
</ParamField>

### Fallback Status Identification

The `fallbackFlag` field can identify whether the task used a fallback model:

- `true`: Generated using fallback model, video resolution is 720p
- `false`: Generated using primary model, may support 1080P (16:9 aspect ratio)

<Note>
  Videos generated using the fallback model cannot be upgraded to high-definition versions through the Get 1080P Video interface.
</Note>

## OpenAPI

```yaml veo3-api/veo3-api.json get /api/v1/veo/record-info
openapi: 3.0.0
info:
  title: Veo3.1 API
  description: kie.ai Veo3.1 API Documentation - Text-to-Video and Image-to-Video API
  version: 1.0.0
  contact:
    name: Technical Support
    email: support@kie.ai
servers:
  - url: https://api.kie.ai
    description: API Server
security:
  - BearerAuth: []
paths:
  /api/v1/veo/record-info:
    get:
      summary: Get Veo3.1 Video Details
      description: Query the execution status and results of Veo3.1 video generation tasks.
      operationId: get-veo3-1-video-details
      parameters:
        - in: query
          name: taskId
          description: Task ID
          required: true
          schema:
            type: string
          example: veo_task_abcdef123456
      responses:
        "200":
          description: Request successful
          content:
            application/json:
              schema:
                type: object
                properties:
                  code:
                    type: integer
                    enum:
                      - 200
                      - 400
                      - 401
                      - 404
                      - 422
                      - 451
                      - 455
                      - 500
                    description: >-
                      Response status code


                      - **200**: Success - Request has been processed
                      successfully

                      - **400**: Your prompt was flagged by Website as violating
                      content policies.

                      Only English prompts are supported at this time.

                      Failed to fetch the image. Kindly verify any access limits
                      set by you or your service provider.

                      public error unsafe image upload.

                      - **401**: Unauthorized - Authentication credentials are
                      missing or invalid

                      - **404**: Not Found - The requested resource or endpoint
                      does not exist

                      - **422**: Validation Error - The request parameters
                      failed validation checks.

                      record is null.

                      Temporarily supports records within 14 days.

                      record result data is blank.

                      record status is not success.

                      record result data not exist.

                      record result data is empty.

                      - **451**: Failed to fetch the image. Kindly verify any
                      access limits set by you or your service provider.

                      - **455**: Service Unavailable - System is currently
                      undergoing maintenance

                      - **500**: Server Error - An unexpected error occurred
                      while processing the request.

                      Timeout

                      Internal Error, Please try again later.
                  msg:
                    type: string
                    description: Error message when code != 200
                    example: success
                  data:
                    type: object
                    properties:
                      taskId:
                        type: string
                        description: Unique identifier of the video generation task
                        example: veo_task_abcdef123456
                      paramJson:
                        type: string
                        description: Request parameters in JSON format
                        example: >-
                          {"prompt":"A futuristic city with flying cars at
                          sunset.","waterMark":"KieAI"}
                      completeTime:
                        type: string
                        format: date-time
                        description: Task completion time
                        example: "2024-03-20T10:30:00Z"
                      response:
                        type: object
                        description: Final result
                        properties:
                          taskId:
                            type: string
                            description: Task ID
                            example: veo_task_abcdef123456
                          resultUrls:
                            type: array
                            items:
                              type: string
                            description: Generated video URLs
                            example:
                              - http://example.com/video1.mp4
                          originUrls:
                            type: array
                            items:
                              type: string
                            description: >-
                              Original video URLs. Only has value when
                              aspectRatio is not 16:9
                            example:
                              - http://example.com/original_video1.mp4
                          resolution:
                            type: string
                            description: Video resolution information
                            example: 1080p
                      successFlag:
                        type: integer
                        description: |-
                          Generation status flag

                          - **0**: Generating
                          - **1**: Success
                          - **2**: Failed
                          - **3**: Generation Failed
                        enum:
                          - 0
                          - 1
                          - 2
                        example: 1
                      errorCode:
                        type: integer
                        format: int32
                        description: >-
                          Error code when task fails


                          - **400**: Your prompt was flagged by Website as
                          violating content policies.

                          Only English prompts are supported at this time.

                          Failed to fetch the image. Kindly verify any access
                          limits set by you or your service provider.

                          public error unsafe image upload.

                          - **500**: Internal Error, Please try again later.

                          Internal Error - Timeout

                          - **501**: Failed - Video generation task failed
                        enum:
                          - 400
                          - 500
                          - 501
                      errorMessage:
                        type: string
                        description: Error message when task fails
                        example: null
                      createTime:
                        type: string
                        format: date-time
                        description: Task creation time
                        example: "2024-03-20T10:25:00Z"
                      fallbackFlag:
                        type: boolean
                        description: >-
                          Whether generated using fallback model. True means
                          backup model was used, false means primary model was
                          used
                        example: false
                        deprecated: true
              example:
                code: 200
                msg: success
                data:
                  taskId: veo_task_abcdef123456
                  paramJson: >-
                    {"prompt":"A futuristic city with flying cars at
                    sunset.","waterMark":"KieAI"}
                  completeTime: "2025-06-06 10:30:00"
                  response:
                    taskId: veo_task_abcdef123456
                    resultUrls:
                      - http://example.com/video1.mp4
                    originUrls:
                      - http://example.com/original_video1.mp4
                    resolution: 1080p
                  successFlag: 1
                  errorCode: null
                  errorMessage: ""
                  createTime: "2025-06-06 10:25:00"
                  fallbackFlag: false
        "500":
          $ref: "#/components/responses/Error"
components:
  responses:
    Error:
      description: Server Error
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: API Key
      description: >-
        All APIs require authentication via Bearer Token.


        Get API Key: 

        1. Visit [API Key Management Page](https://kie.ai/api-key) to get your
        API Key


        Usage:

        Add to request header:

        Authorization: Bearer YOUR_API_KEY
```

---

> To find navigation and other pages in this documentation, fetch the llms.txt file at: https://docs.kie.ai/llms.txt
