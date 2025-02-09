"use client";
import { useEffect, useState } from "react";
import { getAllMedias, uploadMedia } from "@/services/ApiService";
import { useRouter } from "next/navigation";
import { Media } from "@/types";
import { MdFileUpload } from "react-icons/md";
import { IoClose } from "react-icons/io5";
import { VscLoading } from "react-icons/vsc";

export default function Home() {
    const [loading, setLoading] = useState(true);
    const [medias, setMedias] = useState<Media[]>([]);
    const [showMediaPopup, setShowMediaPopup] = useState(false);
    const [fileUploading, setFileUploading] = useState(false);
    const router = useRouter();

    useEffect(() => {
        const token = localStorage.getItem("token");
        if (!token) {
            router.push("/login");
            return;
        }
        getAllMedias(token).then(medias => {
            setMedias(medias);
            setLoading(false);
        }).catch(() => {
            router.push("/login");
        });
    }, [router]);

    const handleFileUpload = async (data: { file: File, name: string }) => {
        setFileUploading(true);
        setShowMediaPopup(false);
        const token = localStorage.getItem("token");
        if (!token) {
            router.push("/login");
            return;
        }

        uploadMedia(data.file, data.name, token).then((media) => {
            setFileUploading(false);
            setMedias([...medias, media]);
        }).catch(() => {
            setFileUploading(false);
        });
    };

    if (loading) {
        return (
            <div className="h-screen flex items-center justify-center">
                <h1>Loading...</h1>
            </div>
        );
    }

    return (
        <>
            <header>
                <h1>Pi Medias</h1>
                <button onClick={() => {
                    localStorage.removeItem("token");
                    router.push("/login");
                }}>Logout</button>
            </header>

            <div className="media-grid">
                {medias.map((media, i) => (
                    <button key={i} className="media-item" onClick={() => router.push("/watch/" + media.id)}>
                        <div className="media-info">
                            <strong>{media.media_name}</strong>
                            <small>{media.mime_type}</small>
                            <small>{new Date(media.created_at).toLocaleDateString()}</small>
                        </div>
                    </button>
                ))}
            </div>

            {fileUploading ? (
                <div className="upload-button">
                    <VscLoading className="animate-spin w-8 h-8" />
                </div>
            ) : (
                <div className="upload-button" onClick={() => setShowMediaPopup(true)}>
                    <MdFileUpload className="w-8 h-8" />
                </div>
            )}

            {showMediaPopup && <MediaPopup onClose={() => setShowMediaPopup(false)} onUpload={handleFileUpload} />}
        </>
    );
}

const MediaPopup = ({ onClose, onUpload }: { onClose: () => void; onUpload: (data: { file: File, name: string }) => void }) => {
    const [name, setName] = useState("");
    const [file, setFile] = useState<File | null>(null);

    return (
        <div className="popup">
            <IoClose className="popup-close" onClick={onClose} />
            <h2>Upload Media</h2>
            <input type="text" placeholder="Name" onChange={(e) => setName(e.target.value)} />
            <input type="file" onChange={(e) => e.target.files && setFile(e.target.files[0])} />
            <button className="primary" onClick={() => file && name && onUpload({ file, name })}>Upload</button>
        </div>
    );
};
