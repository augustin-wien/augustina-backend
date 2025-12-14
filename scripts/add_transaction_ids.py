#!/usr/bin/env python3
"""
Script to add transaction IDs to orders via the API endpoint.
Reads a CSV with order codes and transaction IDs, makes POST requests,
and updates the CSV with success status.
"""

import csv
import sys
import requests
import time
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
import argparse
from typing import List, Dict
import os


def read_csv(file_path: str, delimiter: str = ';') -> tuple[List[str], List[List[str]]]:
    """Read CSV file and return headers and rows."""
    with open(file_path, 'r', encoding='utf-8') as f:
        reader = csv.reader(f, delimiter=delimiter)
        headers = next(reader)
        rows = list(reader)
    return headers, rows


def write_csv(file_path: str, headers: List[str], rows: List[List[str]], delimiter: str = ';'):
    """Write data to CSV file."""
    with open(file_path, 'w', encoding='utf-8', newline='') as f:
        writer = csv.writer(f, delimiter=delimiter)
        writer.writerow(headers)
        writer.writerows(rows)


def add_transaction_id_to_order(
    base_url: str,
    order_code: str,
    transaction_id: str,
    auth_token: str = None,
    session: requests.Session = None
) -> tuple[bool, str]:
    """
    Make POST request to add transaction ID to order.
    
    Returns:
        tuple: (success: bool, message: str)
    """
    url = f"{base_url}/api/orders/unverified/code/{order_code}/transactionID/"
    
    headers = {
        'Content-Type': 'application/json',
        'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Accept-Language': 'en-US,en;q=0.9',
        'Accept': 'application/json',
        'Accept-Encoding': 'gzip, deflate, br',
        'Sec-Fetch-Dest': 'empty',
        'Sec-Fetch-Mode': 'cors',
        'Sec-Fetch-Site': 'same-origin',
    }
    
    if auth_token:
        headers['Authorization'] = f'Bearer {auth_token}'
    
    payload = {
        'transactionID': transaction_id
    }
    
    try:
        # Use provided session or create a new one
        requester = session if session else requests
        response = requester.post(url, json=payload, headers=headers, timeout=10)
        
        if response.status_code == 200:
            return True, "Success"
        elif response.status_code == 401:
            return False, "Unauthorized"
        elif response.status_code == 404:
            return False, "Order not found"
        else:
            return False, f"Error: {response.status_code} - {response.text[:100]}"
    
    except requests.exceptions.Timeout:
        return False, "Timeout"
    except requests.exceptions.ConnectionError:
        return False, "Connection error"
    except Exception as e:
        return False, f"Exception: {str(e)[:100]}"


def ask_continue(row_num: int, order_code: str, error_msg: str) -> bool:
    """
    Ask the user whether to continue processing after a failure.
    Defaults to 'yes' if user just presses Enter.
    
    Returns:
        bool: True to continue, False to stop processing
    """
    print(f"\n{'='*60}")
    print(f"Row {row_num} FAILED for order '{order_code}'")
    print(f"Error: {error_msg}")
    print(f"{'='*60}")
    
    while True:
        response = input("Continue with next row? (Y/n): ").strip().lower()
        if response in ['yes', 'y', '']:  # Empty input defaults to yes
            return True
        elif response in ['no', 'n']:
            return False
        else:
            print("Please enter 'y' or 'n' (or just press Enter to continue)")


def process_csv(
    input_file: str,
    output_file: str,
    base_url: str,
    auth_token: str = None,
    delimiter: str = ';',
    transaction_id_col: str = 'Transaktions-ID',
    order_code_col: str = 'Bestellcode',
    dry_run: bool = False
):
    """
    Process CSV file and add transaction IDs to orders.
    
    Args:
        input_file: Path to input CSV file
        output_file: Path to output CSV file
        base_url: Base URL of the API (e.g., http://localhost:3000)
        auth_token: Optional authentication token
        delimiter: CSV delimiter
        transaction_id_col: Name of transaction ID column
        order_code_col: Name of order code column
        dry_run: If True, don't make actual API calls
    """
    print(f"Reading CSV from: {input_file}")
    headers, rows = read_csv(input_file, delimiter)
    
    # Create a session with retry strategy and proper headers
    session = requests.Session()
    retry_strategy = Retry(
        total=3,
        backoff_factor=1,
        status_forcelist=[429, 500, 502, 503, 504]
    )
    adapter = HTTPAdapter(max_retries=retry_strategy)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    session.headers.update({
        'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Accept-Language': 'en-US,en;q=0.9',
        'Accept': 'application/json',
        'Accept-Encoding': 'gzip, deflate, br',
        'Sec-Fetch-Dest': 'empty',
        'Sec-Fetch-Mode': 'cors',
        'Sec-Fetch-Site': 'same-origin',
    })
    
    # Find column indices
    try:
        transaction_id_idx = headers.index(transaction_id_col)
        order_code_idx = headers.index(order_code_col)
    except ValueError as e:
        print(f"Error: Could not find required column: {e}")
        print(f"Available columns: {', '.join(headers)}")
        sys.exit(1)
    
    # Add new column for API status if it doesn't exist
    status_col_name = 'API_Status'
    if status_col_name not in headers:
        headers.append(status_col_name)
        status_idx = len(headers) - 1
        # Add empty values for existing rows
        for row in rows:
            row.append('')
    else:
        status_idx = headers.index(status_col_name)
    
    # Add new column for error details if it doesn't exist
    error_col_name = 'Error_Details'
    if error_col_name not in headers:
        headers.append(error_col_name)
        error_idx = len(headers) - 1
        # Add empty values for existing rows
        for row in rows:
            row.append('')
    else:
        error_idx = headers.index(error_col_name)
    
    print(f"\nProcessing {len(rows)} rows...")
    print(f"Base URL: {base_url}")
    print(f"Dry run: {dry_run}\n")
    
    success_count = 0
    error_count = 0
    
    for i, row in enumerate(rows, start=2):  # Start at 2 (1 for header + 1-indexed)
        if len(row) <= transaction_id_idx or len(row) <= order_code_idx:
            print(f"Row {i}: Skipping - insufficient columns")
            row.extend([''] * (error_idx + 1 - len(row)))
            row[status_idx] = 'Skipped'
            row[error_idx] = 'Insufficient columns'
            error_count += 1
            continue
        
        transaction_id = row[transaction_id_idx].strip()
        order_code = row[order_code_idx].strip()
        
        if not transaction_id or not order_code:
            print(f"Row {i}: Skipping - missing transaction ID or order code")
            if len(row) <= error_idx:
                row.extend([''] * (error_idx + 1 - len(row)))
            row[status_idx] = 'Skipped'
            row[error_idx] = 'Missing transaction ID or order code'
            error_count += 1
            continue
        
        if dry_run:
            print(f"Row {i}: [DRY RUN] Would add transaction ID '{transaction_id}' to order '{order_code}'")
            if len(row) <= error_idx:
                row.extend([''] * (error_idx + 1 - len(row)))
            row[status_idx] = 'DRY_RUN'
            row[error_idx] = ''
        else:
            success, message = add_transaction_id_to_order(
                base_url, order_code, transaction_id, auth_token, session
            )
            
            # Ensure row has enough columns
            if len(row) <= error_idx:
                row.extend([''] * (error_idx + 1 - len(row)))
            
            if success:
                row[status_idx] = 'Success'
                row[error_idx] = ''
                print(f"Row {i}: ✓ Successfully added transaction ID to order {order_code}")
                success_count += 1
            else:
                row[status_idx] = 'Failed'
                row[error_idx] = message
                print(f"Row {i}: ✗ Failed to add transaction ID to order {order_code}: {message}")
                error_count += 1
                
                # Ask user if they want to continue
                if not ask_continue(i, order_code, message):
                    print("\nStopping processing as requested by user.")
                    session.close()
                    # Write partial results
                    write_csv(output_file, headers, rows, delimiter)
                    print(f"Partial results written to: {output_file}")
                    sys.exit(0)
    
    # Close the session
    session.close()
    
    # Write output CSV
    print(f"\nWriting results to: {output_file}")
    write_csv(output_file, headers, rows, delimiter)
    
    print(f"\n{'='*60}")
    print(f"Summary:")
    print(f"  Total rows processed: {len(rows)}")
    print(f"  Successful: {success_count}")
    print(f"  Failed/Skipped: {error_count}")
    print(f"{'='*60}")


def main():
    parser = argparse.ArgumentParser(
        description='Add transaction IDs to orders via API',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Process CSV with default settings
  python add_transaction_ids.py input.csv output.csv

  # Use custom base URL
  python add_transaction_ids.py input.csv output.csv --base-url http://localhost:3000

  # Use authentication token
  python add_transaction_ids.py input.csv output.csv --token "your-auth-token"

  # Dry run (no actual API calls)
  python add_transaction_ids.py input.csv output.csv --dry-run

  # Use custom column names
  python add_transaction_ids.py input.csv output.csv \\
    --transaction-col "TID" --order-col "OrderCode"
        """
    )
    
    parser.add_argument('input_file', help='Input CSV file path')
    parser.add_argument('output_file', help='Output CSV file path')
    parser.add_argument(
        '--base-url',
        default='http://localhost:3000',
        help='Base URL of the API (default: http://localhost:3000)'
    )
    parser.add_argument(
        '--token',
        help='Authentication token (Bearer token)'
    )
    parser.add_argument(
        '--delimiter',
        default=';',
        help='CSV delimiter (default: ;)'
    )
    parser.add_argument(
        '--transaction-col',
        default='Transaktions-ID',
        help='Name of transaction ID column (default: Transaktions-ID)'
    )
    parser.add_argument(
        '--order-col',
        default='Bestellcode',
        help='Name of order code column (default: Bestellcode)'
    )
    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Perform dry run without making actual API calls'
    )
    
    args = parser.parse_args()
    
    # Check if input file exists
    if not os.path.exists(args.input_file):
        print(f"Error: Input file not found: {args.input_file}")
        sys.exit(1)
    
    # Check if output file already exists and warn
    if os.path.exists(args.output_file):
        response = input(f"Warning: Output file '{args.output_file}' already exists. Overwrite? (y/n): ")
        if response.lower() != 'y':
            print("Aborted.")
            sys.exit(0)
    
    # Get token from environment if not provided
    auth_token = args.token or os.environ.get('AUTH_TOKEN')
    if not auth_token and not args.dry_run:
        print("Warning: No authentication token provided. API calls may fail if authentication is required.")
        print("You can provide a token with --token or set the AUTH_TOKEN environment variable.")
        response = input("Continue anyway? (y/n): ")
        if response.lower() != 'y':
            sys.exit(0)
    
    process_csv(
        input_file=args.input_file,
        output_file=args.output_file,
        base_url=args.base_url,
        auth_token=auth_token,
        delimiter=args.delimiter,
        transaction_id_col=args.transaction_col,
        order_code_col=args.order_col,
        dry_run=args.dry_run
    )


if __name__ == '__main__':
    main()
