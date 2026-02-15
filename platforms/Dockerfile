FROM python:3.13-slim
WORKDIR /app

# Dependencies
COPY platforms/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# bp-agent kurulumu (local wheel)
COPY platforms/bp_agent-*.whl /tmp/
RUN pip install --no-cache-dir /tmp/bp_agent-*.whl && rm /tmp/bp_agent-*.whl

# Platform kaynak kodu
COPY platforms/ .

# Sim modülü
COPY sim/ sim/

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "20000"]
