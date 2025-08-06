# Use distroless static image for minimal footprint
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the pre-built binary
COPY ./main /main

# Set required environment variables
ENV FUNCTION_TARGET=SummarizeArticles
ENV PORT=8080

# Expose port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/main"]