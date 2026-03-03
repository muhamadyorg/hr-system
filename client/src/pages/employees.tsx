import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useLocation } from "wouter";
import { useAuth } from "@/lib/auth";
import { apiRequest, queryClient } from "@/lib/queryClient";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";
import { ArrowLeft, Plus, Search, Users, UserCheck, Loader2 } from "lucide-react";
import type { Employee, Group } from "@shared/schema";

export default function EmployeesPage() {
  const [, navigate] = useLocation();
  const { user } = useAuth();
  const { toast } = useToast();
  const [search, setSearch] = useState("");
  const [showAdd, setShowAdd] = useState(false);
  const [newEmp, setNewEmp] = useState({ employeeNo: "", fullName: "", position: "", groupId: "", phone: "" });

  const { data: employees = [], isLoading } = useQuery<(Employee & { groupName?: string })[]>({
    queryKey: ["/api/employees"],
  });

  const { data: groups = [] } = useQuery<Group[]>({
    queryKey: ["/api/groups"],
  });

  const addMutation = useMutation({
    mutationFn: async (data: any) => {
      const res = await apiRequest("POST", "/api/employees", data);
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["/api/employees"] });
      queryClient.invalidateQueries({ queryKey: ["/api/dashboard/stats"] });
      setShowAdd(false);
      setNewEmp({ employeeNo: "", fullName: "", position: "", groupId: "", phone: "" });
      toast({ title: "Muvaffaqiyatli", description: "Xodim qo'shildi" });
    },
    onError: (err: any) => {
      toast({ title: "Xatolik", description: err.message, variant: "destructive" });
    },
  });

  const filtered = employees.filter(emp =>
    emp.fullName.toLowerCase().includes(search.toLowerCase()) ||
    emp.employeeNo.toLowerCase().includes(search.toLowerCase())
  );

  const handleAdd = () => {
    if (!newEmp.employeeNo.trim() || !newEmp.fullName.trim()) {
      toast({ title: "Xatolik", description: "Xodim raqami va ismi kiritilishi shart", variant: "destructive" });
      return;
    }
    addMutation.mutate({
      employeeNo: newEmp.employeeNo,
      fullName: newEmp.fullName,
      position: newEmp.position || null,
      groupId: newEmp.groupId ? parseInt(newEmp.groupId) : null,
      phone: newEmp.phone || null,
    });
  };

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border sticky top-0 z-50 bg-background">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between gap-2 flex-wrap">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon" onClick={() => navigate("/")} data-testid="button-back">
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <div className="flex items-center gap-2">
              <Users className="w-5 h-5 text-primary" />
              <h1 className="text-lg font-bold text-foreground">Xodimlar</h1>
            </div>
          </div>
          <Button onClick={() => setShowAdd(true)} data-testid="button-add-employee">
            <Plus className="w-4 h-4 mr-1" /> Xodim qo'shish
          </Button>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-6 space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            placeholder="Xodim qidirish..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
            data-testid="input-search-employees"
          />
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          <Badge variant="secondary">{employees.length} ta xodim</Badge>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {[1,2,3,4,5,6].map(i => <Skeleton key={i} className="h-24 rounded-md" />)}
          </div>
        ) : filtered.length === 0 ? (
          <div className="text-center py-16">
            <Users className="w-12 h-12 text-muted-foreground mx-auto mb-3" />
            <p className="text-muted-foreground">
              {search ? "Qidiruv bo'yicha xodim topilmadi" : "Hali xodimlar yo'q"}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {filtered.map((emp) => (
              <Card
                key={emp.id}
                className="hover-elevate active-elevate-2 cursor-pointer overflow-visible"
                onClick={() => navigate(`/employees/${emp.id}`)}
                data-testid={`card-employee-${emp.id}`}
              >
                <CardContent className="p-4 flex items-center gap-3">
                  <Avatar className="w-11 h-11">
                    <AvatarFallback className="bg-primary/10 text-primary font-medium">
                      {emp.fullName.split(" ").map(n => n[0]).join("").slice(0, 2).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-card-foreground truncate">{emp.fullName}</p>
                    <p className="text-xs text-muted-foreground truncate">
                      #{emp.employeeNo} {emp.position ? `· ${emp.position}` : ""}
                    </p>
                    <p className="text-xs text-muted-foreground truncate">
                      {(emp as any).groupName ?? "Guruhsiz"}
                    </p>
                  </div>
                  {emp.hikvisionSynced && (
                    <UserCheck className="w-4 h-4 text-emerald-500 flex-shrink-0" />
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </main>

      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Yangi xodim qo'shish</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Xodim raqami *</Label>
              <Input
                value={newEmp.employeeNo}
                onChange={e => setNewEmp(p => ({ ...p, employeeNo: e.target.value }))}
                placeholder="Masalan: 001"
                data-testid="input-employee-no"
              />
            </div>
            <div className="space-y-2">
              <Label>To'liq ismi *</Label>
              <Input
                value={newEmp.fullName}
                onChange={e => setNewEmp(p => ({ ...p, fullName: e.target.value }))}
                placeholder="Ism familiya"
                data-testid="input-employee-name"
              />
            </div>
            <div className="space-y-2">
              <Label>Lavozimi</Label>
              <Input
                value={newEmp.position}
                onChange={e => setNewEmp(p => ({ ...p, position: e.target.value }))}
                placeholder="Masalan: Dasturchi"
                data-testid="input-employee-position"
              />
            </div>
            <div className="space-y-2">
              <Label>Telefon raqami</Label>
              <Input
                value={newEmp.phone}
                onChange={e => setNewEmp(p => ({ ...p, phone: e.target.value }))}
                placeholder="+998 90 123 45 67"
                data-testid="input-employee-phone"
              />
            </div>
            <div className="space-y-2">
              <Label>Guruh</Label>
              <Select value={newEmp.groupId} onValueChange={v => setNewEmp(p => ({ ...p, groupId: v }))}>
                <SelectTrigger data-testid="select-employee-group">
                  <SelectValue placeholder="Guruhni tanlang" />
                </SelectTrigger>
                <SelectContent>
                  {groups.map(g => (
                    <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setShowAdd(false)}>Bekor qilish</Button>
            <Button onClick={handleAdd} disabled={addMutation.isPending} data-testid="button-confirm-add-employee">
              {addMutation.isPending && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
              Qo'shish
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
