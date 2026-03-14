/** Test: ORM queries → DATABASE_ORM */
import { PrismaClient } from "@prisma/client";

const prisma = new PrismaClient();

export async function getUsers() {
  return prisma.user.findMany({
    where: { active: true },
    include: { posts: true },
  });
}

export async function createUser(email: string, name: string) {
  return prisma.user.create({
    data: { email, name },
  });
}

export async function getUserWithPosts(userId: string) {
  return prisma.user.findUnique({
    where: { id: userId },
    include: { posts: { orderBy: { createdAt: "desc" } } },
  });
}

export async function rawQuery() {
  const result = await prisma.$queryRaw`SELECT * FROM users WHERE active = true`;
  return result;
}
