/** Test: Express/Next.js routes → VIEW_FRAMEWORK */
import express from "express";

const router = express.Router();

export function handleGetUsers(req: any, res: any) {
  res.json({ users: [] });
}

export function authMiddleware(req: any, res: any, next: any) {
  if (!req.headers.authorization) {
    return res.status(401).json({ error: "Unauthorized" });
  }
  next();
}

export function getServerSideProps(context: any) {
  return { props: {} };
}
