import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation } from "wouter";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ArrowLeft, CalendarCheck, Search, Clock, UserCheck, UserX, Filter } from "lucide-react";
import type { Group } from "@shared/schema";

interface AttendanceEntry {
  id: number;
  employeeId: number;
  employeeNo: string;
  fullName: string;
  groupName: string | null;
  eventTime: string;
  status: string;
  photoUrl: string | null;
}

export default function AttendancePage() {
  const [, navigate] = useLocation();
  const [search, setSearch] = useState("");
  const [dateFilter, setDateFilter] = useState(new Date().toISOString().slice(0, 10));
  const [groupFilter, setGroupFilter] = useState("all");

  const { data: records = [], isLoading } = useQuery<AttendanceEntry[]>({
    queryKey: ["/api/attendance", `?date=${dateFilter}${groupFilter !== "all" ? `&groupId=${groupFilter}` : ""}`],
  });

  const { data: groups = [] } = useQuery<Group[]>({
    queryKey: ["/api/groups"],
  });

  const statusLabels: Record<string, string> = {
    check_in: "Keldi",
    check_out: "Ketdi",
    break_out: "Tanaffus",
    break_in: "Qaytdi",
    overtime_in: "Qo'shimcha ish",
    overtime_out: "Qo'shimcha ish tugadi",
  };

  const statusColors: Record<string, string> = {
    check_in: "text-emerald-600 dark:text-emerald-400",
    check_out: "text-blue-600 dark:text-blue-400",
    break_out: "text-amber-600 dark:text-amber-400",
    break_in: "text-violet-600 dark:text-violet-400",
  };

  const filtered = records.filter(r =>
    r.fullName.toLowerCase().includes(search.toLowerCase()) ||
    r.employeeNo.toLowerCase().includes(search.toLowerCase())
  );

  const formatTime = (t: string) => new Date(t).toLocaleTimeString("uz-UZ", { hour: "2-digit", minute: "2-digit", second: "2-digit" });

  const uniqueEmployees = new Set(records.map(r => r.employeeId));
  const checkedIn = new Set(records.filter(r => r.status === "check_in").map(r => r.employeeId));

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border sticky top-0 z-50 bg-background">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" onClick={() => navigate("/")} data-testid="button-back">
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <div className="flex items-center gap-2">
              <CalendarCheck className="w-5 h-5 text-primary" />
              <h1 className="text-lg font-bold text-foreground">Davomat</h1>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-6 space-y-4">
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Xodim qidirish..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="pl-9"
              data-testid="input-search-attendance"
            />
          </div>
          <Input
            type="date"
            value={dateFilter}
            onChange={e => setDateFilter(e.target.value)}
            className="w-auto"
            data-testid="input-date-filter"
          />
          <Select value={groupFilter} onValueChange={setGroupFilter}>
            <SelectTrigger className="w-[180px]" data-testid="select-group-filter">
              <Filter className="w-4 h-4 mr-1" />
              <SelectValue placeholder="Barcha guruhlar" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Barcha guruhlar</SelectItem>
              {groups.map(g => (
                <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-4 flex-wrap">
          <div className="flex items-center gap-1">
            <UserCheck className="w-4 h-4 text-emerald-500" />
            <span className="text-sm text-muted-foreground">{checkedIn.size} keldi</span>
          </div>
          <div className="flex items-center gap-1">
            <Clock className="w-4 h-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">{records.length} ta yozuv</span>
          </div>
        </div>

        {isLoading ? (
          <div className="space-y-3">
            {[1,2,3,4,5].map(i => <Skeleton key={i} className="h-16 rounded-md" />)}
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-16">
            <CalendarCheck className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
            <p className="text-muted-foreground">
              {search ? "Qidiruv bo'yicha natija topilmadi" : "Bu sana uchun davomat ma'lumotlari yo'q"}
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {filtered.map((rec) => (
              <Card key={rec.id} className="overflow-visible" data-testid={`card-attendance-${rec.id}`}>
                <CardContent className="p-3 flex items-center gap-3">
                  <Avatar className="w-10 h-10 flex-shrink-0">
                    <AvatarFallback className="bg-primary/10 text-primary text-sm font-medium">
                      {rec.fullName.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className="text-sm font-semibold text-card-foreground truncate">{rec.fullName}</p>
                      <span className="text-xs text-muted-foreground">#{rec.employeeNo}</span>
                    </div>
                    <p className="text-xs text-muted-foreground truncate">{rec.groupName ?? "Guruhsiz"}</p>
                  </div>
                  <div className="flex items-center gap-2 flex-shrink-0">
                    <Badge variant="secondary" className={`text-xs ${statusColors[rec.status] || ""}`}>
                      {rec.status === "check_in" ? <UserCheck className="w-3 h-3 mr-1" /> : rec.status === "check_out" ? <UserX className="w-3 h-3 mr-1" /> : null}
                      {statusLabels[rec.status] ?? rec.status}
                    </Badge>
                    <span className="text-xs text-muted-foreground font-mono">{formatTime(rec.eventTime)}</span>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
