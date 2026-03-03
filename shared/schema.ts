import { sql } from "drizzle-orm";
import { pgTable, text, varchar, integer, timestamp, boolean, pgEnum } from "drizzle-orm/pg-core";
import { createInsertSchema } from "drizzle-zod";
import { z } from "zod";

export const userRoleEnum = pgEnum("user_role", ["sudo", "admin"]);
export const attendanceStatusEnum = pgEnum("attendance_status", ["check_in", "check_out", "break_out", "break_in", "overtime_in", "overtime_out"]);

export const users = pgTable("users", {
  id: integer("id").primaryKey().generatedAlwaysAsIdentity(),
  username: text("username").notNull().unique(),
  password: text("password").notNull(),
  fullName: text("full_name").notNull(),
  role: userRoleEnum("role").notNull().default("admin"),
  createdBy: integer("created_by"),
  isActive: boolean("is_active").notNull().default(true),
  telegramUserId: text("telegram_user_id"),
});

export const groups = pgTable("groups", {
  id: integer("id").primaryKey().generatedAlwaysAsIdentity(),
  name: text("name").notNull().unique(),
  login: text("login"),
  password: text("password"),
  description: text("description"),
  createdBy: integer("created_by").notNull(),
  assignedAdminId: integer("assigned_admin_id"),
  isActive: boolean("is_active").notNull().default(true),
});

export const employees = pgTable("employees", {
  id: integer("id").primaryKey().generatedAlwaysAsIdentity(),
  employeeNo: text("employee_no").notNull().unique(),
  fullName: text("full_name").notNull(),
  position: text("position"),
  groupId: integer("group_id"),
  photoUrl: text("photo_url"),
  phone: text("phone"),
  isActive: boolean("is_active").notNull().default(true),
  hikvisionSynced: boolean("hikvision_synced").notNull().default(false),
  telegramUserId: text("telegram_user_id"),
});

export const attendanceRecords = pgTable("attendance_records", {
  id: integer("id").primaryKey().generatedAlwaysAsIdentity(),
  employeeId: integer("employee_id").notNull(),
  employeeNo: text("employee_no").notNull(),
  eventTime: timestamp("event_time").notNull(),
  status: attendanceStatusEnum("status").notNull().default("check_in"),
  photoUrl: text("photo_url"),
  deviceIp: text("device_ip"),
  deviceName: text("device_name"),
});

export const adminGroupAccess = pgTable("admin_group_access", {
  id: integer("id").primaryKey().generatedAlwaysAsIdentity(),
  adminId: integer("admin_id").notNull(),
  groupId: integer("group_id").notNull(),
});

export const settings = pgTable("settings", {
  key: text("key").primaryKey(),
  value: text("value").notNull(),
});

// Insert schemas
export const insertUserSchema = createInsertSchema(users).omit({ id: true });
export const insertGroupSchema = createInsertSchema(groups).omit({ id: true });
export const insertEmployeeSchema = createInsertSchema(employees).omit({ id: true });
export const insertAttendanceSchema = createInsertSchema(attendanceRecords).omit({ id: true });

// Login schema
export const loginSchema = z.object({
  username: z.string().min(1, "Foydalanuvchi nomi kiritilishi shart"),
  password: z.string().min(1, "Parol kiritilishi shart"),
});

// Types
export type User = typeof users.$inferSelect;
export type InsertUser = z.infer<typeof insertUserSchema>;
export type Group = typeof groups.$inferSelect;
export type InsertGroup = z.infer<typeof insertGroupSchema>;
export type Employee = typeof employees.$inferSelect;
export type InsertEmployee = z.infer<typeof insertEmployeeSchema>;
export type AttendanceRecord = typeof attendanceRecords.$inferSelect;
export type InsertAttendance = z.infer<typeof insertAttendanceSchema>;
