#!/bin/bash
go-task linux:build
go-task windows:build
go-task linux:create:aur
