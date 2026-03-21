# Bad: Uses implicit :latest tag
FROM ubuntu

# Bad: Explicit :latest tag
FROM python:latest

# Bad: Missing digest verification
FROM node:18-alpine

# Multiple stages with :latest
FROM maven:latest AS builder
FROM openjdk:latest
