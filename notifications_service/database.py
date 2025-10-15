"""
Database connection and operations for notification service.
Uses psycopg2 for PostgreSQL connectivity (similar to lib/pq in Go).
"""

import psycopg2
from psycopg2.extras import RealDictCursor
from datetime import datetime
import os
import uuid


class Database:
    """
    Database class handles all PostgreSQL operations.
    Similar to repository pattern in Go services.
    """
    
    def __init__(self):
        """Initialize database connection using environment variables"""
        self.connection_string = (
            f"host={os.getenv('DB_HOST', 'localhost')} "
            f"port={os.getenv('DB_PORT', '5434')} "
            f"user={os.getenv('DB_USER', 'notifuser')} "
            f"password={os.getenv('DB_PASSWORD', 'notifpass')} "
            f"dbname={os.getenv('DB_NAME', 'notification_db')}"
        )
        self.conn = None
        self.connect()
    
    def connect(self):
        """Establish connection to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(self.connection_string)
            print("✓ Database connection established")
        except Exception as e:
            print(f"❌ Failed to connect to database: {e}")
            raise
    
    def create_notification(self, notification_data: dict) -> dict:
        """
        Insert a new notification record into the database.
        
        Args:
            notification_data: Dictionary containing notification details
            
        Returns:
            Dictionary with created notification including ID
        """
        # Generate unique ID (similar to uuid.New() in Go)
        notification_id = f"notif_{uuid.uuid4()}"
        
        # SQL INSERT query
        query = """
            INSERT INTO notifications (
                id, type, customer_email, customer_id, order_id,
                status, sent_at, error_message, created_at
            ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            RETURNING *
        """
        
        # Current timestamp (like data.Now() in Go)
        now = datetime.utcnow()
        
        try:
            # Get cursor (like preparing a statement in Go)
            cursor = self.conn.cursor(cursor_factory=RealDictCursor)
            
            # Execute query with parameters (prevents SQL injection)
            cursor.execute(query, (
                notification_id,
                notification_data.get('type', 'receipt'),  # Default to 'receipt'
                notification_data['customer_email'],
                notification_data.get('customer_id'),
                notification_data.get('order_id'),
                notification_data.get('status', 'pending'),
                notification_data.get('sent_at'),
                notification_data.get('error_message'),
                now
            ))
            
            # Fetch the inserted row
            result = cursor.fetchone()
            
            # Commit transaction (like tx.Commit() in Go)
            self.conn.commit()
            cursor.close()
            
            return dict(result)
            
        except Exception as e:
            # Rollback on error (like tx.Rollback() in Go)
            self.conn.rollback()
            print(f"[ERROR] Failed to create notification: {e}")
            raise
    
    def update_notification_status(self, notification_id: str, status: str, 
                                   sent_at: datetime = None, error_message: str = None):
        """
        Update notification status after sending attempt.
        
        Args:
            notification_id: ID of the notification
            status: New status (sent/failed)
            sent_at: Timestamp when notification was sent
            error_message: Error message if failed
        """
        query = """
            UPDATE notifications
            SET status = %s, sent_at = %s, error_message = %s
            WHERE id = %s
        """
        
        try:
            cursor = self.conn.cursor()
            cursor.execute(query, (status, sent_at, error_message, notification_id))
            self.conn.commit()
            cursor.close()
        except Exception as e:
            self.conn.rollback()
            print(f"[ERROR] Failed to update notification status: {e}")
            raise
    
    def get_notifications_by_order(self, order_id: str) -> list:
        """
        Retrieve all notifications for a specific order.
        Similar to GetByOrderID methods in Go services.
        
        Args:
            order_id: Order ID to search for
            
        Returns:
            List of notification dictionaries
        """
        query = """
            SELECT * FROM notifications
            WHERE order_id = %s
            ORDER BY created_at DESC
        """
        
        try:
            cursor = self.conn.cursor(cursor_factory=RealDictCursor)
            cursor.execute(query, (order_id,))
            results = cursor.fetchall()
            cursor.close()
            
            # Convert RealDictRow to regular dict
            return [dict(row) for row in results]
            
        except Exception as e:
            print(f"[ERROR] Failed to get notifications: {e}")
            raise
    
    def close(self):
        """Close database connection"""
        if self.conn:
            self.conn.close()
            print("✓ Database connection closed")


# Global database instance (similar to dependency injection in Go)
db = Database()