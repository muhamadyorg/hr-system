import { useQuery, useMutation } from "@tanstack/react-query";
import { useRoute, useLocation } from "wouter";
import { apiRequest, queryClient } from "@/lib/queryClient";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/hooks/use-toast";
import { ArrowLeft, Phone, Briefcase, Users, Clock, UserCheck, UserX, Trash2, Loader2 } from "lucide-react";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog";
import type { Employee, AttendanceRecord } from "@shared/schema";
import { useAuth } from "@/lib/auth";

interface EmployeeDetail extends Employee {
  groupName?: string;
}

export default function EmployeeDetailPage() {
  const [, params] = useRoute("/employees/:id");
  const [, navigate] = useLocation();
  const { toast } = useToast();
  const { user } = useAuth();
  const id = params?.id;

  const { data: employee, isLoading } = useQuery<EmployeeDetail>({
    queryKey: ["/api/employees", id],
    enabled: !!id,
  });

  const { data: attendance = [] } = useQuery<AttendanceRecord[]>({
    queryKey: ["/api/attendance/employee", id],
    enabled: !!id,
  });

  const deleteMutation = useMutation({
    mutationFn: async () => {
      await apiRequest("DELETE", `/api/employees/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/employees"] });
      queryClient.invalidateQueries({ queryKey: ["/api/dashboard/stats"] });
      toast({ title: "Muvaffaqiyatli", description: "Xodim o'chirildi" });
      navigate("/employees");
    },
    onError: (err: any) => {
      toast({ title: "Xatolik", description: err.message, variant: "destructive" });
    },
  });

  const statusLabels: Record<string, string> = {
    check_in: "Keldi",
    check_out: "Ketdi",
    break_out: "Tanaffus",
    break_in: "Qaytdi",
    overtime_in: "Qo'shimcha ish",
    overtime_out: "Qo'shimcha ish tugadi",
  };

  const formatDateTime = (d: string) => {
    const date = new Date(d);
    return date.toLocaleDateString("uz-UZ", { day: "2-digit", month: "2-digit", year: "numeric" }) +
      " " + date.toLocaleTimeString("uz-UZ", { hour: "2-digit", minute: "2-digit" });
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background p-4">
        <Skeleton className="h-8 w-48 mb-6" />
        <Skeleton className="h-64 rounded-md" />
      </div>
    );
  }

  if (!employee) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="text-muted-foreground mb-4">Xodim topilmadi</p>
          <Button onClick={() => navigate("/employees")}>Orqaga</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border sticky top-0 z-50 bg-background">
        <div className="max-w-4xl mx-auto px-4 py-3 flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" onClick={() => navigate("/employees")} data-testid="button-back">
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <h1 className="text-lg font-bold text-foreground">Xodim ma'lumotlari</h1>
          </div>
          {(user?.role === "sudo" || user?.role === "admin") && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive" size="sm" data-testid="button-delete-employee">
                  <Trash2 className="w-4 h-4 mr-1" /> O'chirish
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Xodimni o'chirish</AlertDialogTitle>
                  <AlertDialogDescription>
                    Haqiqatan ham "{employee.fullName}" ni o'chirmoqchimisiz? Bu amalni qaytarib bo'lmaydi.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Bekor qilish</AlertDialogCancel>
                  <AlertDialogAction onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending}>
                    {deleteMutation.isPending && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
                    Ha, o'chirish
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-6 space-y-6">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-start gap-4 flex-wrap">
              <Avatar className="w-16 h-16">
                <AvatarFallback className="bg-primary/10 text-primary text-xl font-bold">
                  {employee.fullName.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div className="flex-1 min-w-0">
                <h2 className="text-xl font-bold text-card-foreground">{employee.fullName}</h2>
                <p className="text-sm text-muted-foreground">Xodim raqami: #{employee.employeeNo}</p>
                <div className="flex items-center gap-2 mt-2 flex-wrap">
                  {employee.hikvisionSynced ? (
                    <Badge variant="secondary" className="text-xs">
                      <UserCheck className="w-3 h-3 mr-1" /> Sinxronlashtirilgan
                    </Badge>
                  ) : (
                    <Badge variant="secondary" className="text-xs">
                      <UserX className="w-3 h-3 mr-1" /> Sinxronlanmagan
                    </Badge>
                  )}
                  {employee.isActive ? (
                    <Badge variant="secondary" className="text-xs text-emerald-600 dark:text-emerald-400">Faol</Badge>
                  ) : (
                    <Badge variant="secondary" className="text-xs text-destructive">Nofaol</Badge>
                  )}
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-6">
              {employee.position && (
                <div className="flex items-center gap-2">
                  <Briefcase className="w-4 h-4 text-muted-foreground" />
                  <div>
                    <p className="text-xs text-muted-foreground">Lavozimi</p>
                    <p className="text-sm font-medium text-foreground">{employee.position}</p>
                  </div>
                </div>
              )}
              <div className="flex items-center gap-2">
                <Users className="w-4 h-4 text-muted-foreground" />
                <div>
                  <p className="text-xs text-muted-foreground">Guruh</p>
                  <p className="text-sm font-medium text-foreground">{employee.groupName ?? "Guruhsiz"}</p>
                </div>
              </div>
              {employee.phone && (
                <div className="flex items-center gap-2">
                  <Phone className="w-4 h-4 text-muted-foreground" />
                  <div>
                    <p className="text-xs text-muted-foreground">Telefon</p>
                    <p className="text-sm font-medium text-foreground">{employee.phone}</p>
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-5">
            <div className="flex items-center gap-2 mb-4">
              <Clock className="w-5 h-5 text-primary" />
              <h3 className="text-base font-semibold text-card-foreground">Davomat tarixi</h3>
            </div>
            {attendance.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8">Davomat ma'lumotlari topilmadi</p>
            ) : (
              <div className="space-y-2">
                {attendance.slice(0, 20).map((rec) => (
                  <div key={rec.id} className="flex items-center justify-between gap-2 p-2 rounded-md bg-muted/30">
                    <div className="flex items-center gap-2 min-w-0">
                      <Badge variant="secondary" className="text-xs flex-shrink-0">
                        {statusLabels[rec.status] ?? rec.status}
                      </Badge>
                    </div>
                    <span className="text-xs text-muted-foreground flex-shrink-0">{formatDateTime(rec.eventTime)}</span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
