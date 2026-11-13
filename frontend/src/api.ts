export const API = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'
export type User = { id:string; displayName:string; email?:string; role:string; languagePair:string; xp?:number; streak?:number; schoolId?:string|null }
export type Task = { taskId:string; fragmentId:string; sourceText:string; suggestedTranslation:string; languagePair:string; status:'pending'|'completed';reward:{baseXp:number} }
export type Progress = { xp:number; level:number; xpInLevel:number; xpForNextLevel:number; streak:number; longestStreak:number; completedTasks:number; confirmedContributions:number }
export type DailyQuest = {code:string;title:string;description:string;goal:number;progress:number;rewardXp:number;completed:boolean;rewarded:boolean}
export type DailyQuestsResponse = {date:string;quests:DailyQuest[]}
export type AnswerResult = {status:string;xpEarned:number;baseXp:number;questBonusXp:number;unlockedQuests:string[];totalXp:number;streak:number}
export type Achievement = {code:string; name:string;description:string;unlocked:boolean;threshold:number}
export type LeaderboardEntry = {rank:number;entityId:string;displayName:string;score:number}
export type Case = {id:string;fragmentId:string;textBa:string;langPair:string;status:string}
export type School = {id:string;name:string}
export type Material = {id:string;title:string;type:string;description:string}
export type Translation = {id:string;textBa:string;translation:string;confirmed:number;status:string;source:{title:string;chapter:string}}
export class ApiError extends Error { constructor(public status:number,message:string){super(message)} }
export async function api<T>(path:string, method='GET', body?:unknown):Promise<T> {
 const token=localStorage.getItem('bashkir_token')
 const headers:Record<string,string>={}
 if(token)headers.Authorization=`Bearer ${token}`
 if(body!==undefined)headers['Content-Type']='application/json'
 let response:Response
 try {response=await fetch(`${API}${path}`, {method,headers,body:body===undefined?undefined:JSON.stringify(body)})}
 catch {throw new ApiError(0,'Нет соединения с сервером. Проверьте, что backend запущен.')}
 if(response.status===204)return undefined as T
 const data=await response.json().catch(()=>({}))
 if(!response.ok)throw new ApiError(response.status,data?.error?.message||`Ошибка ${response.status}`)
 return data as T
}
export function uuid():string {return crypto.randomUUID()}
