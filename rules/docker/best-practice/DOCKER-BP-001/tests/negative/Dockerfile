# Good: Pin specific version
FROM ubuntu:22.04

# Better: Use digest for cryptographic verification
FROM ubuntu:22.04@sha256:b6b83d3c331794420340093eb706a6f152d9c1fa51b262d9bf34594887c2c7ac

# Good: Pin Python minor version
FROM python:3.11.7-slim-bookworm

# Good: Pin Node.js with specific base
FROM node:18.19.0-alpine3.19

# Multi-stage with pinned versions
FROM maven:3.9.6-eclipse-temurin-17 AS builder
FROM eclipse-temurin:17.0.10_7-jre-alpine
