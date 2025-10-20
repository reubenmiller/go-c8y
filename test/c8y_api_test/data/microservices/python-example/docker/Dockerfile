FROM python:3.12-alpine3.22

WORKDIR /app
COPY application.py .
COPY c8y_hservice.py .
RUN pip install flask==0.10.1

ENTRYPOINT ["python"]
CMD ["-u", "application.py"]