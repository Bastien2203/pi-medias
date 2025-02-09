"use client";
import {useParams, useRouter} from 'next/navigation';
import {useEffect, useState} from "react";
import {getMedia} from "@/services/ApiService";
import {Media} from "@/types";


export default function Watch({}) {
    const router = useRouter()
    const {id} = useParams();
    const [media, setMedia] = useState<Media>();

    useEffect(() => {
        const token = localStorage.getItem("token");
        if (!token) {
            router.push("/login");
            return;
        }

        if (!id || typeof id !== "string") {
            router.push("/");
            return;
        }

        getMedia(id, token).then(media => {
            setMedia(media);
        }).catch(() => {
            router.push("/");
        });

    }, [id, router])

    const goHome = () => {
        router.push("/");
    }

    return (
        <div className="">
            <div className="flex justify-between items-center">
                <button onClick={goHome} className="bg-blue-500 text-white px-4 py-2 rounded">Go Home</button>
                <h1 className="text-2xl font-bold">{media?.media_name}</h1>
            </div>
            <div className="mt-4">
                <video src={media?.url} controls className="w-full"></video>
            </div>

        </div>
    );
}
