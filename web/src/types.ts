export type Asset={id:string;canonicalName:string;displayName:string;assetType:string;status:string;environment:string;criticality:string;firstSeenAt:string;lastSeenAt:string};
export type Paged<T>={items:T[];limit:number;offset:number};
export type Overview={assets:number;relationships:number;active:number;stale:number;missing:number;conflicts:number};
export type Claim={connector:string;value:string;authority:string};
export type GraphPayload={nodes:string[];relationships:{id:string;fromAssetId:string;toAssetId:string;type:string;status:string}[];depth:number};
