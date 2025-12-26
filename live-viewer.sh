#!/bin/bash

echo "Live Swap Viewer - Solana Swap Indexer"
echo "======================================="
echo ""

last_sig=""

while true; do
    # Get the latest swap from Redis
    swap=$(docker exec solana-redis redis-cli LINDEX swaps:recent 0 2>/dev/null)
    
    if [ -n "$swap" ]; then
        # Extract signature from JSON
        current_sig=$(echo "$swap" | sed -n 's/.*"signature":"\([^"]*\)".*/\1/p')
        
        # Only display if it's a new swap
        if [ "$current_sig" != "$last_sig" ] && [ -n "$current_sig" ]; then
            # Parse JSON fields
            pair=$(echo "$swap" | sed -n 's/.*"pair":"\([^"]*\)".*/\1/p')
            amount_in=$(echo "$swap" | sed -n 's/.*"amount_in":\([0-9.]*\).*/\1/p')
            token_in=$(echo "$swap" | sed -n 's/.*"token_in":"\([^"]*\)".*/\1/p')
            amount_out=$(echo "$swap" | sed -n 's/.*"amount_out":\([0-9.]*\).*/\1/p')
            token_out=$(echo "$swap" | sed -n 's/.*"token_out":"\([^"]*\)".*/\1/p')
            price=$(echo "$swap" | sed -n 's/.*"price":\([0-9.e+-]*\).*/\1/p')
            
            # Format and display
            printf "[%s] %-20s | %12.4f %-12s -> %12.4f %-12s | Price: %.6f | Sig: %.8s\n" \
                "$(date '+%H:%M:%S')" \
                "$pair" \
                "$amount_in" "$token_in" \
                "$amount_out" "$token_out" \
                "$price" \
                "$current_sig"
            
            last_sig="$current_sig"
        fi
    fi
    
    sleep 1
done
