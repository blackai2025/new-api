# Get 4o Image Details

> Query 4o Image generation task details using taskId, including generation status, parameters and results.

### Status Descriptions

- GENERATING: Generating in progress
- SUCCESS: Generation successful
- CREATE_TASK_FAILED: Failed to create task
- GENERATE_FAILED: Generation failed

### Important Notes

- Maximum query rate is 3 times per second per task

## OpenAPI

```yaml 4o-image-api/4o-image-api.json get /api/v1/gpt4o-image/record-info
openapi: 3.0.0
info:
  title: 4o Image API-GPT Image 1
  description: kie.ai 4o Image API Documentation
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
  /api/v1/gpt4o-image/record-info:
    get:
      summary: Get 4o Image Details
      description: >-
        Query 4o Image generation task details using taskId, including
        generation status, parameters and results.


        ### Status Descriptions

        - GENERATING: Generating in progress

        - SUCCESS: Generation successful

        - CREATE_TASK_FAILED: Failed to create task

        - GENERATE_FAILED: Generation failed


        ### Important Notes

        - Maximum query rate is 3 times per second per task
      operationId: get-4o-image-details
      parameters:
        - in: query
          name: taskId
          description: Unique identifier of the 4o image generation task
          required: true
          schema:
            type: string
          example: task12345
      responses:
        "200":
          description: Request successful
          content:
            application/json:
              schema:
                allOf:
                  - type: object
                    properties:
                      code:
                        type: integer
                        enum:
                          - 200
                          - 401
                          - 402
                          - 404
                          - 422
                          - 429
                          - 455
                          - 500
                        description: >-
                          Response Status Codes


                          - **200**: Success - Request has been processed
                          successfully  

                          - **401**: Unauthorized - Authentication credentials
                          are missing or invalid  

                          - **402**: Insufficient Credits - Account does not
                          have enough credits to perform the operation  

                          - **404**: Not Found - The requested resource or
                          endpoint does not exist  

                          - **422**: Validation Error - The request parameters
                          failed validation checks  

                          - **429**: Rate Limited - Request limit has been
                          exceeded for this resource  

                          - **455**: Service Unavailable - System is currently
                          undergoing maintenance  

                          - **500**: Server Error - An unexpected error occurred
                          while processing the request  
                            - Build Failed - vocal removal generation failed
                      msg:
                        type: string
                        description: Error message when code != 200
                        example: success
                  - type: object
                    properties:
                      data:
                        type: object
                        properties:
                          taskId:
                            type: string
                            description: Unique identifier of the 4o image generation task
                            example: task12345
                          paramJson:
                            type: string
                            description: Request parameters
                            example: >-
                              {"prompt":"A beautiful sunset over the
                              mountains","size":"1:1","isEnhance":false}
                          completeTime:
                            type: integer
                            format: int64
                            description: Task completion time
                            example: 1672574400000
                          response:
                            type: object
                            description: Final result
                            properties:
                              resultUrls:
                                type: array
                                items:
                                  type: string
                                description: List of generated image URLs
                                example:
                                  - https://example.com/result/image1.png
                          successFlag:
                            type: integer
                            format: int32
                            description: Generation status flag
                            example: 1
                          status:
                            type: string
                            description: >-
                              Generation status text, possible values:
                              GENERATING-In progress, SUCCESS-Successful,
                              CREATE_TASK_FAILED-Task creation failed,
                              GENERATE_FAILED-Generation failed


                              - **200**: Success - Image generation completed
                              successfully  

                              - **400**: Bad Request  
                                - The image content in filesUrl violates content policy  
                                - Image size exceeds maximum of 26214400 bytes  
                                - We couldn't process the provided image file (code=invalid_image_format)  
                                - Your content was flagged by OpenAI as violating content policies  
                                - Failed to fetch the image. Kindly verify any access limits set by you or your service provider  
                              - **451**: Download Failed - Unable to download
                              image from the provided filesUrl  

                              - **500**: Internal Error  
                                - Failed to get user token  
                                - Please try again later  
                                - Failed to generate image  
                                - GPT 4O failed to edit the picture  
                                - null
                            enum:
                              - GENERATING
                              - SUCCESS
                              - CREATE_TASK_FAILED
                              - GENERATE_FAILED
                            example: SUCCESS
                          errorCode:
                            type: integer
                            format: int32
                            description: Error code
                            enum:
                              - 200
                              - 400
                              - 451
                              - 500
                          errorMessage:
                            type: string
                            description: Error message
                            example: ""
                          createTime:
                            type: integer
                            format: int64
                            description: Creation time
                            example: 1672561200000
                          progress:
                            type: string
                            description: >-
                              Progress, minimum value is "0.00", maximum value
                              is "1.00"
                            example: "1.00"
              example:
                code: 200
                msg: success
                data:
                  taskId: task12345
                  paramJson: >-
                    {"prompt":"A beautiful sunset over the
                    mountains","size":"1:1","isEnhance":false}
                  completeTime: 1672574400000
                  response:
                    resultUrls:
                      - https://example.com/result/image1.png
                  successFlag: 1
                  status: SUCCESS
                  errorCode: null
                  errorMessage: ""
                  createTime: 1672561200000
                  progress: "1.00"
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


        Note:

        - Keep your API Key secure and do not share it with others

        - If you suspect your API Key has been compromised, reset it immediately
        in the management page
```

---

> To find navigation and other pages in this documentation, fetch the llms.txt file at: https://docs.kie.ai/llms.txt
