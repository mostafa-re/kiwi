#!/bin/bash

# Replication Demo Script
# Demonstrates strong consistency across 1 master + 2 slaves

set -e

MASTER="http://localhost:3300"
SLAVE1="http://localhost:3301"
SLAVE2="http://localhost:3302"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           kiwi Replication Demo                              ║${NC}"
echo -e "${BLUE}║           1 Master + 2 Slaves (Strong Consistency)           ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if cluster is running
echo -e "${YELLOW}[1/7] Checking cluster health...${NC}"
echo ""

echo -n "  Master: "
if curl -s "$MASTER/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not running${NC}"
    echo "Please start the cluster with: make cluster-up"
    exit 1
fi

echo -n "  Slave 1: "
if curl -s "$SLAVE1/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not running${NC}"
fi

echo -n "  Slave 2: "
if curl -s "$SLAVE2/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Running${NC}"
else
    echo -e "${RED}✗ Not running${NC}"
fi
echo ""

# Show cluster status
echo -e "${YELLOW}[2/7] Cluster status from master:${NC}"
curl -s "$MASTER/cluster" | jq .
echo ""

# Write data to master
echo -e "${YELLOW}[3/7] Writing data to MASTER...${NC}"
echo '  PUT {"key": "user:1", "value": {"name": "Alice", "role": "admin"}}'
curl -s -X PUT "$MASTER/objects/" \
    -H "Content-Type: application/json" \
    -d '{"key": "user:1", "value": {"name": "Alice", "role": "admin"}}' | jq .
echo ""

echo '  PUT {"key": "user:2", "value": {"name": "Bob", "role": "user"}}'
curl -s -X PUT "$MASTER/objects/" \
    -H "Content-Type: application/json" \
    -d '{"key": "user:2", "value": {"name": "Bob", "role": "user"}}' | jq .
echo ""

echo '  PUT {"key": "config", "value": {"version": "1.0", "feature_flags": ["dark_mode", "beta"]}}'
curl -s -X PUT "$MASTER/objects/" \
    -H "Content-Type: application/json" \
    -d '{"key": "config", "value": {"version": "1.0", "feature_flags": ["dark_mode", "beta"]}}' | jq .
echo ""

# Read from master
echo -e "${YELLOW}[4/7] Reading from MASTER:${NC}"
echo "  GET user:1"
curl -s "$MASTER/objects/user:1" | jq .
echo ""

# Read from slaves (demonstrating replication)
echo -e "${YELLOW}[5/7] Reading from SLAVE 1 (verify replication):${NC}"
echo "  GET user:1"
curl -s "$SLAVE1/objects/user:1" | jq .
echo ""
echo "  GET user:2"
curl -s "$SLAVE1/objects/user:2" | jq .
echo ""

echo -e "${YELLOW}[6/7] Reading from SLAVE 2 (verify replication):${NC}"
echo "  GET config"
curl -s "$SLAVE2/objects/config" | jq .
echo ""
echo "  LIST all objects"
curl -s "$SLAVE2/objects/" | jq .
echo ""

# Try to write to slave (should fail)
echo -e "${YELLOW}[7/7] Attempting write to SLAVE (should fail - slaves are read-only):${NC}"
echo '  PUT to Slave 1: {"key": "test", "value": "should fail"}'
RESULT=$(curl -s -X PUT "$SLAVE1/objects/" \
    -H "Content-Type: application/json" \
    -d '{"key": "test", "value": "should fail"}')
echo "$RESULT" | jq .
echo ""

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Demo Complete!                            ║${NC}"
echo -e "${BLUE}╠══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║  Summary:                                                    ║${NC}"
echo -e "${BLUE}║  • Writes to master are replicated to all slaves             ║${NC}"
echo -e "${BLUE}║  • Slaves serve reads with consistent data                   ║${NC}"
echo -e "${BLUE}║  • Slaves reject direct writes (read-only)                   ║${NC}"
echo -e "${BLUE}║  • Strong consistency: writes complete only after            ║${NC}"
echo -e "${BLUE}║    all slaves acknowledge                                    ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
