"""
Notification Service - FastAPI microservice for sending email receipts.
Fetches order and payment data, then sends a single receipt email.
"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, EmailStr
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
from starlette.responses import Response
from typing import Optional, Dict, Any
import logging
import json
from datetime import datetime
import os
import requests
import sys
from dotenv import load_dotenv

# Import our custom modules
from database import db
from email_sender import email_sender

# Create FastAPI app instance
app = FastAPI(
    title="Notification Service",
    description="Microservice for sending email receipts",
    version="1.0.0"
)

# Enable CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Service URLs
ORDER_SERVICE_URL = os.getenv("ORDER_SERVICE_URL", "http://order_service:8081")
PAYMENT_SERVICE_URL = os.getenv("PAYMENT_SERVICE_URL", "http://payment_service:8082")


# ============ PYDANTIC MODELS ============

class SendReceiptRequest(BaseModel):
    """Request to send receipt - only needs order_id and customer_email"""
    order_id: str
    customer_email: EmailStr


class NotificationResponse(BaseModel):
    """Standard success response"""
    data: Dict[str, Any]


class JSONFormatter(logging.Formatter):
    def format(self, record):
        log_obj = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "service_name": "notification-service",
            "level": record.levelname,
            "message": record.getMessage(),
            "env": os.getenv("ENVIRONMENT", "development"),
            "host": os.uname().nodename if hasattr(os, 'uname') else "unknown",
            "pid": os.getpid(),
            "trace_id": getattr(record, 'trace_id', None)
        }
        return json.dumps(log_obj)

# Get root logger
logger = logging.getLogger()
logger.setLevel(logging.INFO)
logging.getLogger("uvicorn.access").setLevel(logging.WARNING)

# Remove all existing handlers
for handler in logger.handlers[:]:
    logger.removeHandler(handler)

# Create new handler with JSON formatter
handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(JSONFormatter())
logger.addHandler(handler)

# Configure uvicorn loggers to use our handler
logging.getLogger("uvicorn").handlers = [handler]
logging.getLogger("uvicorn.access").handlers = [handler]
logging.getLogger("uvicorn.error").handlers = [handler]

# ============ Metrics ============

request_count = Counter(
    'http_requests_total', 
    'Total HTTP requests',
    ['method', 'endpoint', 'status']
)

receipts_sent_total = Counter(
    'receipts_sent_total',
    'Total receipts sent'
)

receipts_failed_total = Counter(
    'receipts_failed_total',
    'Total failed receipts'
)

# ============ HELPER FUNCTIONS ============

def fetch_order_details(order_id: str) -> Dict:
    """Fetch order details from order service"""
    try:
        url = f"{ORDER_SERVICE_URL}/orders/{order_id}"
        response = requests.get(url, timeout=5)
        response.raise_for_status()
        
        result = response.json()
        return result.get('data', {})
    
    except requests.RequestException as e:
        print(f"[ERROR] Failed to fetch order: {e}")
        raise HTTPException(status_code=503, detail="Order service unavailable")


def fetch_payment_details(payment_id: str) -> Dict:
    """Fetch payment details from payment service"""
    try:
        url = f"{PAYMENT_SERVICE_URL}/payments/{payment_id}"
        response = requests.get(url, timeout=5)
        response.raise_for_status()
        
        result = response.json()
        return result.get('data', {})
    
    except requests.RequestException as e:
        print(f"[ERROR] Failed to fetch payment: {e}")
        raise HTTPException(status_code=503, detail="Payment service unavailable")


# ============ API ENDPOINTS ============

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "notification-service",
        "database": "connected"
    }

@app.get("/metrics")
async def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

@app.post("/send-receipt", response_model=NotificationResponse)
async def send_receipt(request: SendReceiptRequest):
    """
    Send receipt email for an order.
    
    This endpoint:
    1. Fetches order details from order service
    2. Fetches payment details from payment service
    3. Sends receipt email with all information
    4. Logs notification to database
    """
    logger.info(f"Sending receipt for order {request.order_id} to {request.customer_email}")
    
    try:
        # 1. Fetch order details
        order_data = fetch_order_details(request.order_id)
        
        if not order_data:
            logger.error(f"Failed to fetch order: {e}")
            raise HTTPException(status_code=404, detail="Order not found")
        
        # 2. Fetch payment details (if payment_id exists)
        payment_data = None
        if order_data.get('payment_id'):
            try:
                payment_data = fetch_payment_details(order_data['payment_id'])
            except HTTPException:
                # Payment details not critical, continue without them
                logger.warning("Could not fetch payment details")
        
        # 3. Create notification record in database
        notification_record = {
            'type': 'receipt',
            'customer_email': request.customer_email,
            'customer_id': order_data.get('customer_id'),
            'order_id': request.order_id,
            'status': 'pending'
        }
        
        created_notification = db.create_notification(notification_record)
        notification_id = created_notification['id']
        
        # 4. Send receipt email
        success = email_sender.send_receipt(
            to_email=request.customer_email,
            order_data=order_data,
            payment_data=payment_data
        )
        
        # 5. Update notification status
        if success:
            db.update_notification_status(
                notification_id=notification_id,
                status='sent',
                sent_at=datetime.utcnow()
            )
            
            return NotificationResponse(data={
                "notification_id": notification_id,
                "order_id": request.order_id,
                "status": "sent",
                "sent_at": datetime.utcnow().isoformat()
            })
        else:
            db.update_notification_status(
                notification_id=notification_id,
                status='failed',
                error_message='Email send failed'
            )

            raise HTTPException(status_code=500, detail="Failed to send email")
    
    except HTTPException:
        raise
    
    except Exception as e:
        logger.error(f"[ERROR] Failed to send receipt: {e}")
        raise HTTPException(status_code=500, detail="Failed to send email")


@app.get("/notifications/order/{order_id}")
async def get_order_notifications(order_id: str):
    """Get all notifications for a specific order"""
    try:
        notifications = db.get_notifications_by_order(order_id)
        
        return NotificationResponse(data={
            "order_id": order_id,
            "notifications": notifications,
            "count": len(notifications)
        })
    
    except Exception as e:
        logger.error(f"[ERROR] Failed to get notifications: {e}")
        raise HTTPException(status_code=500, detail="Failed to retrieve notifications")


# ============ STARTUP/SHUTDOWN ============

@app.on_event("startup")
async def startup_event():
    """Service startup"""
    logger.info("===========================================")
    logger.info("📧 Notification Service Starting...")
    logger.info("===========================================")
    logger.info(f"Port: {os.getenv('SERVICE_PORT', '8083')}")
    logger.info(f"Database: {os.getenv('DB_HOST')}:{os.getenv('DB_PORT')}")
    logger.info(f"Order Service: {ORDER_SERVICE_URL}")
    logger.info(f"Payment Service: {PAYMENT_SERVICE_URL}")
    logger.info("-------------------------------------------")
    logger.info("ENDPOINTS:")
    logger.info("  Health:       GET  /health")
    logger.info("  Send Receipt: POST /send-receipt")
    logger.info("  Get Logs:     GET  /notifications/order/{order_id}")
    logger.info("===========================================")


@app.on_event("shutdown")
async def shutdown_event():
    """Service shutdown"""
    logger.info("Shutting down notification service...")
    db.close()
    logger.info("✓ Notification Service shutdown complete")


# ============ RUN SERVER ============

if __name__ == "__main__":
    import uvicorn
    
    port = int(os.getenv('SERVICE_PORT', '8083'))
    
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=port,
        reload=True,
        log_level="info"
    )