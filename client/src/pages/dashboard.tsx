import { useQuery } from "@tanstack/react-query";
import { useLocation } from "wouter";
import { useAuth } from "@/lib/auth";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Users, FolderOpen, CalendarCheck, ShieldCheck, LogOut, Clock, UserCheck, UserX } from "lucide-react";

interface DashboardStats {
  totalEmployees: number;
  totalGroups: number;
  todayPresent: number;
  todayAbsent: number;
  recentAttendance: Array<{
    id: number;
    employeeNo: string;
    fullName: string;
    eventTime: string;
    status: string;
  }>;
  topEmployees: Array<{
    id: number;
    employeeNo: string;
    fullName: string;
    position: string | null;
    groupName: string | null;
    photoUrl: string | null;
  }>;
}

export default function DashboardPage() {
  const [, navigate] = useLocation();
  const { user, logout } = useAuth();

  const { data: stats, isLoading } = useQuery<DashboardStats>({
    queryKey: ["/api/dashboard/stats"],
  });

  const formatTime = (timeStr: string) => {
    const d = new Date(timeStr);
    return d.toLocaleTimeString("uz-UZ", { hour: "2-digit", minute: "2-digit" });
  };

  const statusLabels: Record<string, string> = {
    check_in: "Keldi",
    check_out: "Ketdi",
    break_out: "Tanaffus",
    break_in: "Qaytdi",
    overtime_in: "Qo'shimcha ish",
    overtime_out: "Qo'shimcha ish tugadi",
  };

  const bigButtons = [
    {
      label: "Barcha xodimlar",
      value: stats?.totalEmployees ?? 0,
      icon: Users,
      route: "/employees",
      color: "bg-blue-600 dark:bg-blue-500",
    },
    {
      label: "Guruhlar",
      value: stats?.totalGroups ?? 0,
      icon: FolderOpen,
      route: "/groups",
      color: "bg-emerald-600 dark:bg-emerald-500",
    },
    {
      label: "Bugun kelganlar",
      value: stats?.todayPresent ?? 0,
      icon: CalendarCheck,
      route: "/attendance",
      color: "bg-violet-600 dark:bg-violet-500",
    },
    {
      label: user?.role === "sudo" ? "Sudo panel" : "Admin panel",
      value: user?.role === "sudo" ? "Boshqaruv" : "Sozlamalar",
      icon: ShieldCheck,
      route: user?.role === "sudo" ? "/sudo" : "/admin-settings",
      color: "bg-amber-600 dark:bg-amber-500",
    },
  ];

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border sticky top-0 z-50 bg-background">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-md bg-primary flex items-center justify-center">
              <ShieldCheck className="w-5 h-5 text-primary-foreground" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-foreground leading-tight">HR Tizimi</h1>
              <p className="text-xs text-muted-foreground">Hikvision Davomat</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="secondary" className="text-xs">
              {user?.role === "sudo" ? "SUDO" : "ADMIN"}
            </Badge>
            <span className="text-sm text-foreground">{user?.fullName}</span>
            <Button variant="ghost" size="icon" onClick={logout} data-testid="button-logout">
              <LogOut className="w-4 h-4" />
            </Button>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-6 space-y-6">
        {isLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {[1,2,3,4].map(i => (
              <Skeleton key={i} className="h-36 rounded-md" />
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {bigButtons.map((btn) => (
              <button
                key={btn.route}
                onClick={() => navigate(btn.route)}
                className="group relative overflow-visible rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
                data-testid={`button-nav-${btn.route.replace("/", "")}`}
              >
                <Card className="h-full hover-elevate active-elevate-2 overflow-visible">
                  <CardContent className="p-6 flex items-center gap-4">
                    <div className={`w-14 h-14 rounded-md ${btn.color} flex items-center justify-center flex-shrink-0`}>
                      <btn.icon className="w-7 h-7 text-white" />
                    </div>
                    <div className="text-left">
                      <p className="text-2xl font-bold text-card-foreground">{btn.value}</p>
                      <p className="text-sm text-muted-foreground">{btn.label}</p>
                    </div>
                  </CardContent>
                </Card>
              </button>
            ))}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card>
            <CardContent className="p-5">
              <div className="flex items-center justify-between gap-2 mb-4 flex-wrap">
                <h3 className="text-base font-semibold text-card-foreground">Bugungi davomat</h3>
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1">
                    <UserCheck className="w-4 h-4 text-emerald-500" />
                    <span className="text-sm text-muted-foreground">{stats?.todayPresent ?? 0} keldi</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <UserX className="w-4 h-4 text-destructive" />
                    <span className="text-sm text-muted-foreground">{stats?.todayAbsent ?? 0} kelmadi</span>
                  </div>
                </div>
              </div>
              {isLoading ? (
                <div className="space-y-3">
                  {[1,2,3].map(i => <Skeleton key={i} className="h-12" />)}
                </div>
              ) : stats?.recentAttendance?.length ? (
                <div className="space-y-2">
                  {stats.recentAttendance.slice(0, 8).map((rec) => (
                    <div key={rec.id} className="flex items-center justify-between gap-2 p-2 rounded-md bg-muted/30">
                      <div className="flex items-center gap-2 min-w-0">
                        <Clock className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                        <span className="text-sm font-medium text-foreground truncate">{rec.fullName}</span>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <Badge variant="secondary" className="text-xs">
                          {statusLabels[rec.status] ?? rec.status}
                        </Badge>
                        <span className="text-xs text-muted-foreground">{formatTime(rec.eventTime)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground text-center py-8">Bugun hali davomat qayd etilmagan</p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-5">
              <div className="flex items-center justify-between gap-2 mb-4 flex-wrap">
                <h3 className="text-base font-semibold text-card-foreground">Xodimlar</h3>
                <Button variant="ghost" size="sm" onClick={() => navigate("/employees")} data-testid="button-see-all-employees">
                  Barchasini ko'rish
                </Button>
              </div>
              {isLoading ? (
                <div className="space-y-3">
                  {[1,2,3,4,5].map(i => <Skeleton key={i} className="h-12" />)}
                </div>
              ) : stats?.topEmployees?.length ? (
                <div className="space-y-2">
                  {stats.topEmployees.slice(0, 5).map((emp) => (
                    <div
                      key={emp.id}
                      className="flex items-center gap-3 p-2 rounded-md hover-elevate cursor-pointer"
                      onClick={() => navigate(`/employees/${emp.id}`)}
                      data-testid={`card-employee-${emp.id}`}
                    >
                      <Avatar className="w-9 h-9">
                        <AvatarFallback className="bg-primary/10 text-primary text-sm font-medium">
                          {emp.fullName.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium text-foreground truncate">{emp.fullName}</p>
                        <p className="text-xs text-muted-foreground truncate">
                          {emp.groupName ?? "Guruhsiz"} {emp.position ? `· ${emp.position}` : ""}
                        </p>
                      </div>
                      <span className="text-xs text-muted-foreground flex-shrink-0">#{emp.employeeNo}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground text-center py-8">Xodimlar topilmadi</p>
              )}
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  );
}
