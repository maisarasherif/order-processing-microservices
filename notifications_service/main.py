"""
Notification Service - FastAPI microservice for sending email receipts.
Fetches order and payment data, then sends a single receipt email.
"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, EmailStr
from typing import Optional, Dict, Any
from datetime import datetime
import os
import requests
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
    print(f"📧 Sending receipt for order {request.order_id} to {request.customer_email}")
    
    try:
        # 1. Fetch order details
        order_data = fetch_order_details(request.order_id)
        
        if not order_data:
            raise HTTPException(status_code=404, detail="Order not found")
        
        # 2. Fetch payment details (if payment_id exists)
        payment_data = None
        if order_data.get('payment_id'):
            try:
                payment_data = fetch_payment_details(order_data['payment_id'])
            except HTTPException:
                # Payment details not critical, continue without them
                print(f"[WARNING] Could not fetch payment details")
        
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
        print(f"[ERROR] Failed to send receipt: {e}")
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
        print(f"[ERROR] Failed to get notifications: {e}")
        raise HTTPException(status_code=500, detail="Failed to retrieve notifications")


# ============ STARTUP/SHUTDOWN ============

@app.on_event("startup")
async def startup_event():
    """Service startup"""
    print("===========================================")
    print("📧 Notification Service Starting...")
    print("===========================================")
    print(f"Port: {os.getenv('SERVICE_PORT', '8083')}")
    print(f"Database: {os.getenv('DB_HOST')}:{os.getenv('DB_PORT')}")
    print(f"Order Service: {ORDER_SERVICE_URL}")
    print(f"Payment Service: {PAYMENT_SERVICE_URL}")
    print("-------------------------------------------")
    print("ENDPOINTS:")
    print("  Health:       GET  /health")
    print("  Send Receipt: POST /send-receipt")
    print("  Get Logs:     GET  /notifications/order/{order_id}")
    print("===========================================")


@app.on_event("shutdown")
async def shutdown_event():
    """Service shutdown"""
    print("Shutting down notification service...")
    db.close()
    print("✓ Notification Service shutdown complete")


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