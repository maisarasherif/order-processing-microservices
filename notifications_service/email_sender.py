"""
Email sending functionality using SMTP.
Sends receipt emails with order and payment information.
"""

import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
from jinja2 import Environment, FileSystemLoader
import os
from typing import Optional, Dict, Any
import logging

logger = logging.getLogger(__name__)


class EmailSender:
    """Handles SMTP email operations"""
    
    def __init__(self):
        """Initialize email sender with SMTP configuration"""
        self.smtp_host = os.getenv('SMTP_HOST', 'smtp.gmail.com')
        self.smtp_port = int(os.getenv('SMTP_PORT', '587'))
        self.smtp_user = os.getenv('SMTP_USER')
        self.smtp_password = os.getenv('SMTP_PASSWORD')
        
        self.from_email = os.getenv('FROM_EMAIL', 'noreply@foodorder.com')
        self.from_name = os.getenv('FROM_NAME', 'Food Order System')
        
        # Setup Jinja2 template engine
        self.template_env = Environment(
            loader=FileSystemLoader('templates'),
            autoescape=True
        )
    
    def send_receipt(self, to_email: str, order_data: Dict[str, Any], 
                     payment_data: Optional[Dict[str, Any]] = None) -> bool:
        """
        Send receipt email with order and payment details.
        
        Args:
            to_email: Customer's email
            order_data: Order information from order service
            payment_data: Payment information from payment service (optional)
            
        Returns:
            True if sent successfully
        """
        try:
            # Load and render template
            template = self.template_env.get_template('receipt.html')
            html_content = template.render(
                order=order_data,
                payment=payment_data
            )
            
            subject = f"Receipt for Order #{order_data.get('id', 'N/A')}"
            
            return self._send_email(to_email, subject, html_content)
            
        except Exception as e:
            logger.error(f"[ERROR] Failed to send receipt: {e}")
            return False
    
    def _send_email(self, to_email: str, subject: str, html_content: str) -> bool:
        """Internal method to send email via SMTP"""
        try:
            # Create MIME message
            message = MIMEMultipart('alternative')
            message['Subject'] = subject
            message['From'] = f"{self.from_name} <{self.from_email}>"
            message['To'] = to_email
            
            # Attach HTML content
            html_part = MIMEText(html_content, 'html')
            message.attach(html_part)
            
            # Connect and send
            with smtplib.SMTP(self.smtp_host, self.smtp_port) as server:
                server.starttls()
                server.login(self.smtp_user, self.smtp_password)
                server.send_message(message)
            
            logger.info(f"✓ Email sent to {to_email}: {subject}")
            return True
            
        except Exception as e:
            logger.error(f"[ERROR] SMTP error: {e}")
            return False


# Global email sender instance
email_sender = EmailSender()